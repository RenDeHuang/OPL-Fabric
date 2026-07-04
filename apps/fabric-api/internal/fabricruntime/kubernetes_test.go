package fabricruntime

import (
	"context"
	"errors"
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

func TestKubernetesRuntimeCreatesDedicatedNodePoolBeforeCompute(t *testing.T) {
	client := fake.NewSimpleClientset()
	capacity := &recordingCapacity{nodePoolID: "np-1"}
	runtime := KubernetesRuntime{Provider: fabrick8s.Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest", StorageClassName: "cbs"}, Capacity: capacity}

	result, err := runtime.CreateCompute(context.Background(), postgres.ComputeResourceRow{
		ID:                   "compute-1",
		IsolationMode:        "dedicated_nodepool",
		ComputeShapeJSON:     `{"cpu":4,"memoryGb":8}`,
		ProviderInstanceType: "SA5.LARGE8",
	})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}
	if result.NodePoolID != "np-1" {
		t.Fatalf("NodePoolID = %q, want np-1", result.NodePoolID)
	}
	if capacity.createdComputeID != "compute-1" || capacity.verifiedNodePoolID != "np-1" {
		t.Fatalf("capacity = %+v", capacity)
	}
}

func TestKubernetesRuntimeSkipsCapacityForSharedPoolCompute(t *testing.T) {
	client := fake.NewSimpleClientset()
	capacity := &recordingCapacity{err: errors.New("should_not_call_capacity")}
	runtime := KubernetesRuntime{Provider: fabrick8s.Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest", StorageClassName: "cbs"}, Capacity: capacity}

	if _, err := runtime.CreateCompute(context.Background(), postgres.ComputeResourceRow{ID: "compute-1", IsolationMode: "shared_pool"}); err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}
	if capacity.createdComputeID != "" {
		t.Fatalf("capacity should not be called: %+v", capacity)
	}
}

func TestKubernetesRuntimeReusesExistingDedicatedNodePoolID(t *testing.T) {
	client := fake.NewSimpleClientset()
	capacity := &recordingCapacity{nodePoolID: "np-new"}
	runtime := KubernetesRuntime{Provider: fabrick8s.Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest", StorageClassName: "cbs"}, Capacity: capacity}

	result, err := runtime.CreateCompute(context.Background(), postgres.ComputeResourceRow{
		ID:            "compute-1",
		IsolationMode: "dedicated_nodepool",
		NodePoolID:    "np-existing",
	})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}
	if result.NodePoolID != "np-existing" {
		t.Fatalf("NodePoolID = %q, want np-existing", result.NodePoolID)
	}
	if capacity.createdComputeID != "" {
		t.Fatalf("existing nodepool should be reused without ensure: %+v", capacity)
	}
	if capacity.verifiedNodePoolID != "np-existing" {
		t.Fatalf("verified nodepool = %q, want np-existing", capacity.verifiedNodePoolID)
	}
}

func TestKubernetesRuntimeDeletesDedicatedNodePoolWhenDestroyingCompute(t *testing.T) {
	client := fake.NewSimpleClientset()
	capacity := &recordingCapacity{}
	runtime := KubernetesRuntime{Provider: fabrick8s.Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest", StorageClassName: "cbs"}, Capacity: capacity}
	compute, err := runtime.Provider.CreateCompute(context.Background(), fabrick8s.CreateComputeInput{ID: "compute-1"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	if err := runtime.DestroyCompute(context.Background(), postgres.ComputeResourceRow{ID: "compute-1", ProviderRef: compute.ProviderRef, RuntimeRef: compute.ServiceRef, NodePoolID: "np-1"}); err != nil {
		t.Fatalf("DestroyCompute: %v", err)
	}
	if capacity.deletedNodePoolID != "np-1" {
		t.Fatalf("deleted nodepool = %q", capacity.deletedNodePoolID)
	}
}

type recordingCapacity struct {
	nodePoolID         string
	err                error
	createdComputeID   string
	verifiedNodePoolID string
	deletedNodePoolID  string
}

func (c *recordingCapacity) EnsureNodePool(_ context.Context, req CapacityNodePoolRequest) (CapacityNodePoolResult, error) {
	if c.err != nil {
		return CapacityNodePoolResult{}, c.err
	}
	c.createdComputeID = req.ComputeID
	return CapacityNodePoolResult{NodePoolID: c.nodePoolID}, nil
}

func (c *recordingCapacity) VerifyNodePool(_ context.Context, nodePoolID string) (bool, error) {
	if c.err != nil {
		return false, c.err
	}
	c.verifiedNodePoolID = nodePoolID
	return true, nil
}

func (c *recordingCapacity) DeleteNodePool(_ context.Context, nodePoolID string) error {
	if c.err != nil {
		return c.err
	}
	c.deletedNodePoolID = nodePoolID
	return nil
}
