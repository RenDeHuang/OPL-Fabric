package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
)

func TestApplyStorageVolumeOperationCreatesPVCAndMarksSucceeded(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-1"] = postgres.OperationRow{ID: "op-1", ResourceKind: "storage_volume", ResourceID: "storage-1", State: "accepted"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", OwnerAccountID: "acct-1", SizeGB: 10, State: "creating"}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime}

	receipt, err := orch.Apply(context.Background(), "op-1")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if receipt.State != "succeeded" {
		t.Fatalf("state = %q, want succeeded", receipt.State)
	}
	if store.operations["op-1"].State != "succeeded" {
		t.Fatalf("operation state = %q", store.operations["op-1"].State)
	}
	if store.storage["storage-1"].ProviderRef != "pvc/storage-1" || store.storage["storage-1"].State != "available" {
		t.Fatalf("storage row = %+v", store.storage["storage-1"])
	}
	if runtime.createdStorageID != "storage-1" {
		t.Fatalf("created storage = %q", runtime.createdStorageID)
	}
}

func TestApplyComputeOperationCreatesRuntimeAndMarksSucceeded(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-1"] = postgres.OperationRow{ID: "op-1", ResourceKind: "compute_allocation", ResourceID: "compute-1", State: "accepted"}
	store.compute["compute-1"] = postgres.ComputeAllocationRow{ID: "compute-1", OwnerAccountID: "acct-1", State: "creating", ComputeShapeJSON: `{"cpu":2,"memoryGb":4}`}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime}

	receipt, err := orch.Apply(context.Background(), "op-1")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if receipt.State != "succeeded" {
		t.Fatalf("state = %q, want succeeded", receipt.State)
	}
	if store.compute["compute-1"].ProviderRef != "deployment/compute-1" || store.compute["compute-1"].RuntimeRef != "service/compute-1" || store.compute["compute-1"].State != "running" {
		t.Fatalf("compute row = %+v", store.compute["compute-1"])
	}
	if runtime.createdComputeAllocationID != "compute-1" {
		t.Fatalf("created compute = %q", runtime.createdComputeAllocationID)
	}
}

func TestApplyComputeOperationPersistsNodePoolID(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-1"] = postgres.OperationRow{ID: "op-1", ResourceKind: "compute_allocation", ResourceID: "compute-1", State: "accepted"}
	store.compute["compute-1"] = postgres.ComputeAllocationRow{ID: "compute-1", OwnerAccountID: "acct-1", State: "creating", IsolationMode: "workspace_exclusive_cvm"}
	runtime := &recordingRuntime{nodePoolID: "np-1"}
	orch := Orchestrator{Store: store, Runtime: runtime}

	if _, err := orch.Apply(context.Background(), "op-1"); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if store.compute["compute-1"].NodePoolID != "np-1" {
		t.Fatalf("NodePoolID = %q, want np-1", store.compute["compute-1"].NodePoolID)
	}
}

func TestApplyAttachmentOperationMountsStorage(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-1"] = postgres.OperationRow{ID: "op-1", ResourceKind: "storage_attachment", ResourceID: "attach-1", State: "accepted"}
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", OwnerAccountID: "acct-1", ComputeAllocationID: "compute-1", StorageID: "storage-1", State: "attaching", MountPath: "/data"}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime}

	_, err := orch.Apply(context.Background(), "op-1")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if store.attachments["attach-1"].ProviderRef != "mount/attach-1" || store.attachments["attach-1"].State != "attached" {
		t.Fatalf("attachment row = %+v", store.attachments["attach-1"])
	}
	if runtime.attachedID != "attach-1" {
		t.Fatalf("attached id = %q", runtime.attachedID)
	}
}

func TestApplyWorkspaceEntryOperationCreatesGatewayEntry(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-1"] = postgres.OperationRow{ID: "op-1", ResourceKind: "workspace_entry", ResourceID: "entry-1", State: "accepted"}
	store.entries["entry-1"] = postgres.WorkspaceEntryRow{ID: "entry-1", OwnerAccountID: "acct-1", WorkspaceID: "ws-1", AttachmentID: "attach-1", State: "creating", Host: "workspace.medopl.cn", Path: "/w/ws-1/"}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime}

	_, err := orch.Apply(context.Background(), "op-1")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if store.entries["entry-1"].State != "ready" {
		t.Fatalf("entry row = %+v", store.entries["entry-1"])
	}
	if runtime.workspaceEntryID != "entry-1" {
		t.Fatalf("workspace entry id = %q", runtime.workspaceEntryID)
	}
}

func TestApplyWorkspaceOperationRunsStorageComputeAttachmentEntryInOrder(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-workspace"] = postgres.OperationRow{ID: "op-workspace", ResourceKind: "workspace", ResourceID: "ws-1", State: "accepted"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", OwnerAccountID: "acct-1", SizeGB: 20, State: "creating", Retained: true}
	store.compute["compute-1"] = postgres.ComputeAllocationRow{ID: "compute-1", OwnerAccountID: "acct-1", State: "creating", ComputeShapeJSON: `{"cpu":2,"memoryGb":4}`}
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", OwnerAccountID: "acct-1", ComputeAllocationID: "compute-1", StorageID: "storage-1", State: "attaching", MountPath: "/data"}
	store.entries["entry-1"] = postgres.WorkspaceEntryRow{ID: "entry-1", OwnerAccountID: "acct-1", WorkspaceID: "ws-1", AttachmentID: "attach-1", State: "creating", Host: "workspace.medopl.cn", Path: "/w/ws-1/"}
	store.workspaces["ws-1"] = postgres.WorkspaceRow{ID: "ws-1", OwnerAccountID: "acct-1", StorageID: "storage-1", ComputeAllocationID: "compute-1", AttachmentID: "attach-1", EntryID: "entry-1", OperationID: "op-workspace", State: "provisioning"}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime}

	receipt, err := orch.Apply(context.Background(), "op-workspace")
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if receipt.State != "succeeded" || store.operations["op-workspace"].State != "succeeded" {
		t.Fatalf("receipt=%+v operation=%+v", receipt, store.operations["op-workspace"])
	}
	if store.storage["storage-1"].State != "available" || store.storage["storage-1"].ProviderRef != "pvc/storage-1" {
		t.Fatalf("storage row = %+v", store.storage["storage-1"])
	}
	if store.compute["compute-1"].State != "running" || store.compute["compute-1"].ProviderRef != "deployment/compute-1" || store.compute["compute-1"].RuntimeRef != "service/compute-1" {
		t.Fatalf("compute row = %+v", store.compute["compute-1"])
	}
	if store.attachments["attach-1"].State != "attached" || store.attachments["attach-1"].ProviderRef != "mount/attach-1" {
		t.Fatalf("attachment row = %+v", store.attachments["attach-1"])
	}
	if store.entries["entry-1"].State != "ready" {
		t.Fatalf("entry row = %+v", store.entries["entry-1"])
	}
	if store.entries["entry-1"].ServiceRef != "service/compute-1" || runtime.workspaceEntryServiceRef != "service/compute-1" {
		t.Fatalf("entry service ref row=%q runtime=%q", store.entries["entry-1"].ServiceRef, runtime.workspaceEntryServiceRef)
	}
	if store.workspaces["ws-1"].State != "running" {
		t.Fatalf("workspace row = %+v", store.workspaces["ws-1"])
	}
	if got := runtime.calls; len(got) != 4 || got[0] != "storage:storage-1" || got[1] != "compute:compute-1" || got[2] != "attach:attach-1" || got[3] != "entry:entry-1" {
		t.Fatalf("runtime calls = %#v", got)
	}
}

func TestApplyWorkspaceOperationResolvesComputePoolBeforeCreatingCompute(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-workspace"] = postgres.OperationRow{ID: "op-workspace", ResourceKind: "workspace", ResourceID: "ws-1", State: "accepted"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", OwnerAccountID: "acct-1", SizeGB: 20, State: "creating", Retained: true}
	store.compute["compute-1"] = postgres.ComputeAllocationRow{
		ID:                   "compute-1",
		OwnerAccountID:       "acct-1",
		ProductPresetID:      "basic",
		State:                "creating",
		ComputeShapeJSON:     `{"cpu":2,"memoryGb":4}`,
		ProviderInstanceType: "SA5.LARGE4",
		CapacityPoolID:       "tencent-cpu-compute-pool",
		IsolationMode:        "workspace_exclusive_cvm",
	}
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", OwnerAccountID: "acct-1", ComputeAllocationID: "compute-1", StorageID: "storage-1", State: "attaching", MountPath: "/data"}
	store.entries["entry-1"] = postgres.WorkspaceEntryRow{ID: "entry-1", OwnerAccountID: "acct-1", WorkspaceID: "ws-1", AttachmentID: "attach-1", State: "creating", Host: "workspace.medopl.cn", Path: "/w/ws-1/"}
	store.workspaces["ws-1"] = postgres.WorkspaceRow{ID: "ws-1", OwnerAccountID: "acct-1", StorageID: "storage-1", ComputeAllocationID: "compute-1", AttachmentID: "attach-1", EntryID: "entry-1", OperationID: "op-workspace", State: "provisioning"}
	runtime := &recordingRuntime{}
	resolver := &recordingComputePoolResolver{nodePoolID: "np-basic-2c4g"}
	orch := Orchestrator{Store: store, Runtime: runtime, ComputePools: resolver}

	if _, err := orch.Apply(context.Background(), "op-workspace"); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if resolver.computeAllocationID != "compute-1" || resolver.workspaceID != "ws-1" || resolver.providerInstanceType != "SA5.LARGE4" {
		t.Fatalf("resolver = %+v", resolver)
	}
	if store.compute["compute-1"].NodePoolID != "np-basic-2c4g" {
		t.Fatalf("stored node pool id = %q", store.compute["compute-1"].NodePoolID)
	}
	if runtime.createdNodePoolID != "np-basic-2c4g" {
		t.Fatalf("runtime node pool id = %q", runtime.createdNodePoolID)
	}
	if store.workspaces["ws-1"].State != "running" {
		t.Fatalf("workspace row = %+v", store.workspaces["ws-1"])
	}
}

func TestApplyWorkspaceOperationFailsWhenComputePoolResolverReturnsNoNodePoolID(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-workspace"] = postgres.OperationRow{ID: "op-workspace", ResourceKind: "workspace", ResourceID: "ws-1", State: "accepted"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", OwnerAccountID: "acct-1", SizeGB: 20, State: "creating", Retained: true}
	store.compute["compute-1"] = postgres.ComputeAllocationRow{
		ID:                   "compute-1",
		OwnerAccountID:       "acct-1",
		ProductPresetID:      "basic",
		State:                "creating",
		ComputeShapeJSON:     `{"cpu":2,"memoryGb":4}`,
		ProviderInstanceType: "SA5.LARGE4",
		CapacityPoolID:       "tencent-cpu-compute-pool",
		IsolationMode:        "workspace_exclusive_cvm",
	}
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", OwnerAccountID: "acct-1", ComputeAllocationID: "compute-1", StorageID: "storage-1", State: "attaching", MountPath: "/data"}
	store.entries["entry-1"] = postgres.WorkspaceEntryRow{ID: "entry-1", OwnerAccountID: "acct-1", WorkspaceID: "ws-1", AttachmentID: "attach-1", State: "creating", Host: "workspace.medopl.cn", Path: "/w/ws-1/"}
	store.workspaces["ws-1"] = postgres.WorkspaceRow{ID: "ws-1", OwnerAccountID: "acct-1", StorageID: "storage-1", ComputeAllocationID: "compute-1", AttachmentID: "attach-1", EntryID: "entry-1", OperationID: "op-workspace", State: "provisioning"}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime, ComputePools: &recordingComputePoolResolver{}}

	_, err := orch.Apply(context.Background(), "op-workspace")
	if err == nil || err.Error() != "compute_pool_not_resolved" {
		t.Fatalf("Apply error = %v, want compute_pool_not_resolved", err)
	}
	if runtime.createdComputeAllocationID != "" {
		t.Fatalf("compute must not be created before compute pool is resolved: %+v", runtime)
	}
}

func TestApplyWorkspaceOperationFailureLeavesStorageRetained(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-workspace"] = postgres.OperationRow{ID: "op-workspace", ResourceKind: "workspace", ResourceID: "ws-1", State: "accepted"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", OwnerAccountID: "acct-1", SizeGB: 20, State: "creating", Retained: true}
	store.compute["compute-1"] = postgres.ComputeAllocationRow{ID: "compute-1", OwnerAccountID: "acct-1", State: "creating"}
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", OwnerAccountID: "acct-1", ComputeAllocationID: "compute-1", StorageID: "storage-1", State: "attaching", MountPath: "/data"}
	store.entries["entry-1"] = postgres.WorkspaceEntryRow{ID: "entry-1", OwnerAccountID: "acct-1", WorkspaceID: "ws-1", AttachmentID: "attach-1", State: "creating"}
	store.workspaces["ws-1"] = postgres.WorkspaceRow{ID: "ws-1", StorageID: "storage-1", ComputeAllocationID: "compute-1", AttachmentID: "attach-1", EntryID: "entry-1", OperationID: "op-workspace", State: "provisioning"}
	runtimeErr := errors.New("compute_failed")
	orch := Orchestrator{Store: store, Runtime: &recordingRuntime{computeErr: runtimeErr}}

	_, err := orch.Apply(context.Background(), "op-workspace")
	if !errors.Is(err, runtimeErr) {
		t.Fatalf("error = %v, want %v", err, runtimeErr)
	}
	if store.operations["op-workspace"].State != "failed" {
		t.Fatalf("operation state = %q, want failed", store.operations["op-workspace"].State)
	}
	if store.storage["storage-1"].State != "available" || store.storage["storage-1"].ProviderRef == "" || !store.storage["storage-1"].Retained {
		t.Fatalf("storage should remain available and retained: %+v", store.storage["storage-1"])
	}
	if store.compute["compute-1"].State != "creating" {
		t.Fatalf("compute row = %+v", store.compute["compute-1"])
	}
	if store.workspaces["ws-1"].State != "provisioning" {
		t.Fatalf("workspace row = %+v", store.workspaces["ws-1"])
	}
}

func TestApplyDestroyOperationsCallRuntimeWithoutDeletingRetainedStorage(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-compute"] = postgres.OperationRow{ID: "op-compute", ResourceKind: "compute_allocation_destroy", ResourceID: "compute-1", State: "accepted"}
	store.operations["op-storage"] = postgres.OperationRow{ID: "op-storage", ResourceKind: "storage_destroy", ResourceID: "storage-1", State: "accepted"}
	store.compute["compute-1"] = postgres.ComputeAllocationRow{ID: "compute-1", State: "running", ProviderRef: "deployment/compute-1", RuntimeRef: "service/compute-1", NodePoolID: "np-1"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", State: "available", ProviderRef: "pvc/storage-1", Retained: true}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime}

	if _, err := orch.Apply(context.Background(), "op-compute"); err != nil {
		t.Fatalf("Apply compute destroy: %v", err)
	}
	if _, err := orch.Apply(context.Background(), "op-storage"); err != nil {
		t.Fatalf("Apply storage destroy: %v", err)
	}

	if store.compute["compute-1"].State != "destroyed" || runtime.destroyedComputeAllocationID != "compute-1" {
		t.Fatalf("compute destroy failed: row=%+v runtime=%q", store.compute["compute-1"], runtime.destroyedComputeAllocationID)
	}
	if store.compute["compute-1"].ProviderRef != "" || store.compute["compute-1"].RuntimeRef != "" || store.compute["compute-1"].NodePoolID != "" {
		t.Fatalf("compute refs should be cleared for rebuild: %+v", store.compute["compute-1"])
	}
	if store.storage["storage-1"].State != "available" || runtime.destroyedStorageID != "" {
		t.Fatalf("retained storage should not be destroyed: row=%+v runtime=%q", store.storage["storage-1"], runtime.destroyedStorageID)
	}
}

func TestApplyAttachmentDetachOperationCallsRuntime(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-detach"] = postgres.OperationRow{ID: "op-detach", ResourceKind: "attachment_detach", ResourceID: "attach-1", State: "accepted"}
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", State: "attached", ProviderRef: "deployment/compute-1:pvc/storage-1"}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime}

	if _, err := orch.Apply(context.Background(), "op-detach"); err != nil {
		t.Fatalf("Apply attachment detach: %v", err)
	}

	if store.attachments["attach-1"].State != "detached" || runtime.detachedID != "attach-1" {
		t.Fatalf("detach failed: row=%+v runtime=%q", store.attachments["attach-1"], runtime.detachedID)
	}
}

func TestApplyMarksOperationFailedWhenRuntimeFails(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-1"] = postgres.OperationRow{ID: "op-1", ResourceKind: "storage_volume", ResourceID: "storage-1", State: "accepted"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", SizeGB: 10, State: "creating"}
	runtimeErr := errors.New("runtime_failed")
	orch := Orchestrator{Store: store, Runtime: &recordingRuntime{err: runtimeErr}}

	_, err := orch.Apply(context.Background(), "op-1")
	if !errors.Is(err, runtimeErr) {
		t.Fatalf("error = %v, want %v", err, runtimeErr)
	}
	if store.operations["op-1"].State != "failed" {
		t.Fatalf("operation state = %q, want failed", store.operations["op-1"].State)
	}
}

type memoryStore struct {
	operations  map[string]postgres.OperationRow
	storage     map[string]postgres.StorageVolumeRow
	compute     map[string]postgres.ComputeAllocationRow
	attachments map[string]postgres.StorageAttachmentRow
	entries     map[string]postgres.WorkspaceEntryRow
	workspaces  map[string]postgres.WorkspaceRow
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		operations:  map[string]postgres.OperationRow{},
		storage:     map[string]postgres.StorageVolumeRow{},
		compute:     map[string]postgres.ComputeAllocationRow{},
		attachments: map[string]postgres.StorageAttachmentRow{},
		entries:     map[string]postgres.WorkspaceEntryRow{},
		workspaces:  map[string]postgres.WorkspaceRow{},
	}
}

func (s *memoryStore) GetOperation(_ context.Context, id string) (postgres.OperationRow, error) {
	return s.operations[id], nil
}

func (s *memoryStore) UpdateOperationState(_ context.Context, id, state string) error {
	row := s.operations[id]
	row.State = state
	s.operations[id] = row
	return nil
}

func (s *memoryStore) GetStorageVolume(_ context.Context, id string) (postgres.StorageVolumeRow, error) {
	return s.storage[id], nil
}

func (s *memoryStore) UpdateStorageVolume(_ context.Context, row postgres.StorageVolumeRow) error {
	s.storage[row.ID] = row
	return nil
}

func (s *memoryStore) GetComputeAllocation(_ context.Context, id string) (postgres.ComputeAllocationRow, error) {
	return s.compute[id], nil
}

func (s *memoryStore) UpdateComputeAllocation(_ context.Context, row postgres.ComputeAllocationRow) error {
	s.compute[row.ID] = row
	return nil
}

func (s *memoryStore) GetStorageAttachment(_ context.Context, id string) (postgres.StorageAttachmentRow, error) {
	return s.attachments[id], nil
}

func (s *memoryStore) UpdateStorageAttachment(_ context.Context, row postgres.StorageAttachmentRow) error {
	s.attachments[row.ID] = row
	return nil
}

func (s *memoryStore) GetWorkspaceEntry(_ context.Context, id string) (postgres.WorkspaceEntryRow, error) {
	return s.entries[id], nil
}

func (s *memoryStore) UpdateWorkspaceEntry(_ context.Context, row postgres.WorkspaceEntryRow) error {
	s.entries[row.ID] = row
	return nil
}

func (s *memoryStore) GetWorkspace(_ context.Context, id string) (postgres.WorkspaceRow, error) {
	return s.workspaces[id], nil
}

func (s *memoryStore) UpdateWorkspace(_ context.Context, row postgres.WorkspaceRow) error {
	s.workspaces[row.ID] = row
	return nil
}

type recordingRuntime struct {
	err                          error
	computeErr                   error
	nodePoolID                   string
	calls                        []string
	createdStorageID             string
	createdComputeAllocationID   string
	createdNodePoolID            string
	attachedID                   string
	workspaceEntryID             string
	workspaceEntryServiceRef     string
	destroyedComputeAllocationID string
	destroyedStorageID           string
	detachedID                   string
}

func (r *recordingRuntime) CreateStorageVolume(_ context.Context, row postgres.StorageVolumeRow) (RuntimeStorageResult, error) {
	if r.err != nil {
		return RuntimeStorageResult{}, r.err
	}
	r.calls = append(r.calls, "storage:"+row.ID)
	r.createdStorageID = row.ID
	return RuntimeStorageResult{ProviderRef: "pvc/" + row.ID}, nil
}

func (r *recordingRuntime) CreateCompute(_ context.Context, row postgres.ComputeAllocationRow) (RuntimeComputeResult, error) {
	if r.err != nil {
		return RuntimeComputeResult{}, r.err
	}
	if r.computeErr != nil {
		return RuntimeComputeResult{}, r.computeErr
	}
	r.calls = append(r.calls, "compute:"+row.ID)
	r.createdComputeAllocationID = row.ID
	r.createdNodePoolID = row.NodePoolID
	return RuntimeComputeResult{ProviderRef: "deployment/" + row.ID, RuntimeRef: "service/" + row.ID, NodePoolID: r.nodePoolID}, nil
}

func (r *recordingRuntime) AttachStorage(_ context.Context, row postgres.StorageAttachmentRow) (RuntimeAttachmentResult, error) {
	if r.err != nil {
		return RuntimeAttachmentResult{}, r.err
	}
	r.calls = append(r.calls, "attach:"+row.ID)
	r.attachedID = row.ID
	return RuntimeAttachmentResult{ProviderRef: "mount/" + row.ID}, nil
}

func (r *recordingRuntime) CreateWorkspaceEntry(_ context.Context, row postgres.WorkspaceEntryRow) error {
	if r.err != nil {
		return r.err
	}
	r.calls = append(r.calls, "entry:"+row.ID)
	r.workspaceEntryID = row.ID
	r.workspaceEntryServiceRef = row.ServiceRef
	return nil
}

func (r *recordingRuntime) DestroyCompute(_ context.Context, row postgres.ComputeAllocationRow) error {
	if r.err != nil {
		return r.err
	}
	r.destroyedComputeAllocationID = row.ID
	return nil
}

func (r *recordingRuntime) DestroyStorage(_ context.Context, row postgres.StorageVolumeRow) error {
	if r.err != nil {
		return r.err
	}
	r.destroyedStorageID = row.ID
	return nil
}

func (r *recordingRuntime) DetachStorage(_ context.Context, row postgres.StorageAttachmentRow) error {
	if r.err != nil {
		return r.err
	}
	r.detachedID = row.ID
	return nil
}

type recordingComputePoolResolver struct {
	nodePoolID           string
	computeAllocationID  string
	workspaceID          string
	providerInstanceType string
}

func (r *recordingComputePoolResolver) ResolveComputePool(_ context.Context, req ComputePoolRequest) (ComputePoolResult, error) {
	r.computeAllocationID = req.ComputeAllocationID
	r.workspaceID = req.WorkspaceID
	r.providerInstanceType = req.ProviderInstanceType
	return ComputePoolResult{NodePoolID: r.nodePoolID}, nil
}
