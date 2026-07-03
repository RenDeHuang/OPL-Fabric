package k8s

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type Provider struct {
	Client         kubernetes.Interface
	Namespace      string
	WorkspaceImage string
}

type CreateComputeInput struct {
	ID            string
	WorkspaceName string
	PackageID     string
	CPU           int
	MemoryGB      int
}

type CreateComputeResult struct {
	ProviderRef string
	ServiceRef  string
}

func (p Provider) CreateCompute(ctx context.Context, input CreateComputeInput) (CreateComputeResult, error) {
	name := k8sName(input.ID)
	computeKey := labelValue(input.ID)
	labels := map[string]string{
		"app.kubernetes.io/name":     "opl-workspace",
		"app.kubernetes.io/instance": name,
		"oplcloud.cn/compute-key":    computeKey,
	}
	annotations := map[string]string{
		"oplcloud.cn/compute-id": input.ID,
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.Namespace, Labels: labels, Annotations: annotations},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr[int32](1),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr(false),
					Containers: []corev1.Container{{
						Name:  "workspace",
						Image: p.WorkspaceImage,
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 3000}},
						Env: []corev1.EnvVar{
							{Name: "OPL_COMPUTE_ID", Value: input.ID},
							{Name: "OPL_WORKSPACE_NAME", Value: input.WorkspaceName},
							{Name: "OPL_PACKAGE_ID", Value: input.PackageID},
						},
					}},
				},
			},
		},
	}
	if _, err := p.Client.AppsV1().Deployments(p.Namespace).Create(ctx, deploy, metav1.CreateOptions{}); err != nil {
		return CreateComputeResult{}, err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.Namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Name: "http", Port: 3000, TargetPort: intstr.FromInt(3000)}},
		},
	}
	if _, err := p.Client.CoreV1().Services(p.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		_ = p.Client.AppsV1().Deployments(p.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
		return CreateComputeResult{}, err
	}

	return CreateComputeResult{ProviderRef: "deployment/" + name, ServiceRef: "service/" + name}, nil
}

func k8sName(id string) string {
	return boundedName("opl", id, 63)
}

func labelValue(id string) string {
	return boundedName("compute", id, 63)
}

func boundedName(prefix, id string, limit int) string {
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return '-'
	}, id)
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "resource"
	}
	hash := shortHash(id)
	maxClean := limit - len(prefix) - len(hash) - 2
	if maxClean < 1 {
		maxClean = 1
	}
	if len(clean) > maxClean {
		clean = strings.Trim(clean[:maxClean], "-")
	}
	if clean == "" {
		clean = "resource"
	}
	return fmt.Sprintf("%s-%s-%s", prefix, clean, hash)
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func ptr[T any](value T) *T {
	return &value
}
