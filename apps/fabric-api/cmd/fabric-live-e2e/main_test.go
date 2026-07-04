package main

import (
	"context"
	"strings"
	"testing"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestVerifyKubernetesInputsBootstrapsFabricNamespaceAndPullSecret(t *testing.T) {
	t.Setenv("OPL_LIVE_E2E_BOOTSTRAP_NAMESPACE", "true")
	t.Setenv("OPL_LIVE_E2E_BOOTSTRAP_IMAGE_PULL_SECRET", "true")
	t.Setenv("OPL_LIVE_E2E_IMAGE_PULL_SECRET_SOURCE_NAMESPACE", "opl-cloud")
	client := fake.NewSimpleClientset(
		&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "cbs"}},
		&networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "qcloud"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "opl-cloud"}},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "tcr-pull-secret", Namespace: "opl-cloud"},
			Type:       corev1.SecretTypeDockerConfigJson,
			Data:       map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{}}`)},
		},
	)
	cfg := config.Config{
		KubernetesNamespace: "oplfabric",
		StorageClass:        "cbs",
		IngressClass:        "qcloud",
		ImagePullSecretName: "tcr-pull-secret",
	}

	if err := verifyKubernetesInputs(context.Background(), client, cfg); err != nil {
		t.Fatalf("verifyKubernetesInputs: %v", err)
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "oplfabric", metav1.GetOptions{}); err != nil {
		t.Fatalf("namespace was not created: %v", err)
	}
	if _, err := client.CoreV1().ServiceAccounts("oplfabric").Get(context.Background(), "opl-fabric-api", metav1.GetOptions{}); err != nil {
		t.Fatalf("service account was not created: %v", err)
	}
	if _, err := client.RbacV1().Roles("oplfabric").Get(context.Background(), "opl-fabric-api", metav1.GetOptions{}); err != nil {
		t.Fatalf("role was not created: %v", err)
	}
	if _, err := client.RbacV1().RoleBindings("oplfabric").Get(context.Background(), "opl-fabric-api", metav1.GetOptions{}); err != nil {
		t.Fatalf("role binding was not created: %v", err)
	}
	if _, err := client.CoreV1().Secrets("oplfabric").Get(context.Background(), "tcr-pull-secret", metav1.GetOptions{}); err != nil {
		t.Fatalf("image pull secret was not copied: %v", err)
	}
}

func TestVerifyKubernetesInputsDoesNotCreateNamespaceWithoutBootstrapFlag(t *testing.T) {
	client := fake.NewSimpleClientset(
		&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "cbs"}},
		&networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "qcloud"}},
	)
	cfg := config.Config{
		KubernetesNamespace: "oplfabric",
		StorageClass:        "cbs",
		IngressClass:        "qcloud",
		ImagePullSecretName: "tcr-pull-secret",
	}

	err := verifyKubernetesInputs(context.Background(), client, cfg)
	if err == nil {
		t.Fatal("verifyKubernetesInputs should fail when namespace is absent and bootstrap is disabled")
	}
	if _, getErr := client.CoreV1().Namespaces().Get(context.Background(), "oplfabric", metav1.GetOptions{}); !apierrors.IsNotFound(getErr) {
		t.Fatalf("namespace should not be created, get err=%v", getErr)
	}
}

func TestWaitDeploymentAvailableIncludesKubernetesDiagnosticsOnTimeout(t *testing.T) {
	replicas := int32(1)
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "opl-compute-timeout", Namespace: "oplfabric"},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "workspace"}},
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				UpdatedReplicas:   1,
				ReadyReplicas:     0,
				AvailableReplicas: 0,
				Conditions: []appsv1.DeploymentCondition{{
					Type:    appsv1.DeploymentAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "MinimumReplicasUnavailable",
					Message: "Deployment does not have minimum availability",
				}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "workspace-pod", Namespace: "oplfabric", Labels: map[string]string{"app": "workspace"}},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				ContainerStatuses: []corev1.ContainerStatus{{
					Name: "workspace",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{
						Reason:  "ImagePullBackOff",
						Message: "failed to pull image",
					}},
				}},
			},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "workspace-pod.1", Namespace: "oplfabric"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "workspace-pod", Namespace: "oplfabric"},
			Type:           corev1.EventTypeWarning,
			Reason:         "Failed",
			Message:        "Failed to pull image",
		},
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitDeploymentAvailable(ctx, client, "oplfabric", "opl-compute-timeout")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	message := err.Error()
	for _, want := range []string{
		"deployment_status name=opl-compute-timeout",
		"condition Available=False reason=MinimumReplicasUnavailable",
		"pod name=workspace-pod phase=Pending",
		"container workspace waiting=ImagePullBackOff",
		"event Warning Failed pod/workspace-pod: Failed to pull image",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error missing %q:\n%s", want, message)
		}
	}
}
