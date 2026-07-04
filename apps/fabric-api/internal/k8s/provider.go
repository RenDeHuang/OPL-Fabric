package k8s

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type Provider struct {
	Client               kubernetes.Interface
	Namespace            string
	WorkspaceImage       string
	WorkspaceWebUIPort   int32
	WorkspaceDataDir     string
	WorkspaceProjectsDir string
	CodexHome            string
	CodexModel           string
	CodexReasoningEffort string
	CodexBaseURL         string
	CodexAPIKey          string
	CodexModelProvider   string
	CodexProviderName    string
}

type CreateComputeInput struct {
	ID                   string
	WorkspaceName        string
	ProductPresetID      string
	ComputeShapeJSON     string
	ProviderInstanceType string
	CapacityPoolID       string
	IsolationMode        string
	NodePoolID           string
	RuntimeRef           string
	CPU                  int
	MemoryGB             int
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
	if input.CapacityPoolID != "" {
		annotations["oplcloud.cn/capacity-pool-id"] = input.CapacityPoolID
	}
	if input.IsolationMode != "" {
		annotations["oplcloud.cn/isolation-mode"] = input.IsolationMode
	}
	if input.NodePoolID != "" {
		annotations["oplcloud.cn/node-pool-id"] = input.NodePoolID
	}
	if input.RuntimeRef != "" {
		annotations["oplcloud.cn/runtime-ref"] = input.RuntimeRef
	}
	if input.ProviderInstanceType != "" {
		annotations["oplcloud.cn/provider-instance-type"] = input.ProviderInstanceType
	}
	codexSecretName := ""
	if secret := p.codexSecret(name, labels); secret != nil {
		if _, err := p.Client.CoreV1().Secrets(p.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return CreateComputeResult{}, err
		}
		codexSecretName = secret.Name
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
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: p.workspaceWebUIPort()}},
						Resources: corev1.ResourceRequirements{
							Requests: resourceList(input),
							Limits:   resourceList(input),
						},
						EnvFrom: p.codexEnvFrom(codexSecretName),
						Env: []corev1.EnvVar{
							{Name: "OPL_COMPUTE_ID", Value: input.ID},
							{Name: "OPL_WORKSPACE_NAME", Value: input.WorkspaceName},
							{Name: "OPL_PRODUCT_PRESET_ID", Value: input.ProductPresetID},
							{Name: "OPL_COMPUTE_SHAPE_JSON", Value: input.ComputeShapeJSON},
							{Name: "OPL_CAPACITY_POOL_ID", Value: input.CapacityPoolID},
							{Name: "OPL_ISOLATION_MODE", Value: input.IsolationMode},
							{Name: "OPL_PROJECTS_DIR", Value: defaultString(p.WorkspaceProjectsDir, "/projects")},
							{Name: "OPL_WEBUI_AUTH_MODE", Value: "none"},
							{Name: "OPL_WORKSPACE_ROOT", Value: defaultString(p.WorkspaceProjectsDir, "/projects")},
							{Name: "CODEX_HOME", Value: defaultString(p.CodexHome, "/data/codex")},
						},
					}},
				},
			},
		},
	}
	if _, err := p.Client.AppsV1().Deployments(p.Namespace).Create(ctx, deploy, metav1.CreateOptions{}); err != nil {
		if codexSecretName != "" {
			if deleteErr := p.Client.CoreV1().Secrets(p.Namespace).Delete(ctx, codexSecretName, metav1.DeleteOptions{}); deleteErr != nil {
				return CreateComputeResult{}, errors.Join(err, fmt.Errorf("cleanup secret %q: %w", codexSecretName, deleteErr))
			}
		}
		return CreateComputeResult{}, err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.Namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Name: "http", Port: p.workspaceWebUIPort(), TargetPort: intstr.FromInt(int(p.workspaceWebUIPort()))}},
		},
	}
	if _, err := p.Client.CoreV1().Services(p.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		if deleteErr := p.Client.AppsV1().Deployments(p.Namespace).Delete(ctx, name, metav1.DeleteOptions{}); deleteErr != nil {
			return CreateComputeResult{}, errors.Join(err, fmt.Errorf("cleanup deployment %q: %w", name, deleteErr))
		}
		if codexSecretName != "" {
			if deleteErr := p.Client.CoreV1().Secrets(p.Namespace).Delete(ctx, codexSecretName, metav1.DeleteOptions{}); deleteErr != nil {
				return CreateComputeResult{}, errors.Join(err, fmt.Errorf("cleanup secret %q: %w", codexSecretName, deleteErr))
			}
		}
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

func resourceList(input CreateComputeInput) corev1.ResourceList {
	resources := corev1.ResourceList{}
	if input.CPU > 0 {
		resources[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%d", input.CPU))
	}
	if input.MemoryGB > 0 {
		resources[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dGi", input.MemoryGB))
	}
	return resources
}

func (p Provider) workspaceWebUIPort() int32 {
	if p.WorkspaceWebUIPort > 0 {
		return p.WorkspaceWebUIPort
	}
	return 3000
}

func (p Provider) codexSecret(name string, labels map[string]string) *corev1.Secret {
	data := map[string][]byte{}
	put := func(key, value string) {
		if value != "" {
			data[key] = []byte(value)
		}
	}
	put("OPL_CODEX_MODEL", p.CodexModel)
	put("OPL_CODEX_REASONING_EFFORT", p.CodexReasoningEffort)
	put("OPL_CODEX_BASE_URL", p.CodexBaseURL)
	put("OPL_CODEX_API_KEY", p.CodexAPIKey)
	put("OPL_CODEX_MODEL_PROVIDER", p.CodexModelProvider)
	put("OPL_CODEX_PROVIDER_NAME", p.CodexProviderName)
	if len(data) == 0 {
		return nil
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-env", Namespace: p.Namespace, Labels: labels},
		Type:       corev1.SecretTypeOpaque,
		Data:       data,
	}
}

func (p Provider) codexEnvFrom(secretName string) []corev1.EnvFromSource {
	if secretName == "" {
		return nil
	}
	return []corev1.EnvFromSource{{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
		},
	}}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func ptr[T any](value T) *T {
	return &value
}
