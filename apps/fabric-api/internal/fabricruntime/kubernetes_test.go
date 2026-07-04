package fabricruntime

import (
	"context"
	"strings"
	"testing"

	fabrick8s "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/k8s"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/orchestrator"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesRuntimeImplementsOrchestratorRuntime(t *testing.T) {
	var _ orchestrator.Runtime = KubernetesRuntime{}
}

func TestKubernetesRuntimeCreatesStorageAndComputeFromRows(t *testing.T) {
	client := fake.NewSimpleClientset()
	runtime := KubernetesRuntime{Provider: fabrick8s.Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest", StorageClassName: "cbs"}}

	storage, err := runtime.CreateStorageVolume(context.Background(), postgres.StorageVolumeRow{ID: "storage-1", SizeGB: 10})
	if err != nil {
		t.Fatalf("CreateStorageVolume: %v", err)
	}
	if !strings.HasPrefix(storage.ProviderRef, "pvc/") {
		t.Fatalf("storage result = %+v", storage)
	}

	compute, err := runtime.CreateCompute(context.Background(), postgres.ComputeResourceRow{ID: "compute-1", ComputeShapeJSON: `{"cpu":2,"memoryGb":4}`})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}
	if !strings.HasPrefix(compute.ProviderRef, "deployment/") || !strings.HasPrefix(compute.RuntimeRef, "service/") {
		t.Fatalf("compute result = %+v", compute)
	}
}

func TestKubernetesRuntimeAttachesAndDestroysFromRows(t *testing.T) {
	client := fake.NewSimpleClientset()
	runtime := KubernetesRuntime{Provider: fabrick8s.Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest", StorageClassName: "cbs"}}
	compute, err := runtime.Provider.CreateCompute(context.Background(), fabrick8s.CreateComputeInput{ID: "compute-1"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}
	storage, err := runtime.Provider.CreateStorageVolume(context.Background(), fabrick8s.CreateStorageVolumeInput{ID: "storage-1", SizeGB: 10})
	if err != nil {
		t.Fatalf("CreateStorageVolume: %v", err)
	}

	attachment, err := runtime.AttachStorage(context.Background(), postgres.StorageAttachmentRow{ID: "attach-1", ComputeID: "compute-1", StorageID: "storage-1", MountPath: "/data", ProviderRef: compute.ProviderRef + ":" + storage.ProviderRef})
	if err != nil {
		t.Fatalf("AttachStorage: %v", err)
	}
	if !strings.Contains(attachment.ProviderRef, ":pvc/") {
		t.Fatalf("attachment result = %+v", attachment)
	}
	if err := runtime.DestroyCompute(context.Background(), postgres.ComputeResourceRow{ID: "compute-1", ProviderRef: compute.ProviderRef, RuntimeRef: compute.ServiceRef}); err != nil {
		t.Fatalf("DestroyCompute: %v", err)
	}
	if _, err := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), strings.TrimPrefix(compute.ProviderRef, "deployment/"), metav1.GetOptions{}); err == nil {
		t.Fatal("deployment should be deleted")
	}
}
