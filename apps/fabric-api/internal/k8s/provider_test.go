package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateComputeCreatesDeploymentAndService(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            "compute-1",
		WorkspaceName: "Alpha",
		PackageID:     "basic",
		CPU:           2,
		MemoryGB:      4,
	})
	if err != nil {
		t.Fatalf("create compute failed: %v", err)
	}
	if result.ProviderRef != "deployment/opl-compute-1" {
		t.Fatalf("provider ref = %s", result.ProviderRef)
	}

	deploy, err := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), "opl-compute-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	if deploy.Spec.Template.Spec.Containers[0].Image != "workspace-image:latest" {
		t.Fatalf("image mismatch")
	}

	_, err = client.CoreV1().Services("opl-cloud").Get(context.Background(), "opl-compute-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("service missing: %v", err)
	}
}

var _ = appsv1.Deployment{}
var _ = corev1.Service{}
