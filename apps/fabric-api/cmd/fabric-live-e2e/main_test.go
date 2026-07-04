package main

import (
	"context"
	"testing"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/config"
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
