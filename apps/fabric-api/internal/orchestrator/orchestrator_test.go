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
	store.operations["op-1"] = postgres.OperationRow{ID: "op-1", ResourceKind: "compute_resource", ResourceID: "compute-1", State: "accepted"}
	store.compute["compute-1"] = postgres.ComputeResourceRow{ID: "compute-1", OwnerAccountID: "acct-1", State: "creating", ComputeShapeJSON: `{"cpu":2,"memoryGb":4}`}
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
	if runtime.createdComputeID != "compute-1" {
		t.Fatalf("created compute = %q", runtime.createdComputeID)
	}
}

func TestApplyComputeOperationPersistsNodePoolID(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-1"] = postgres.OperationRow{ID: "op-1", ResourceKind: "compute_resource", ResourceID: "compute-1", State: "accepted"}
	store.compute["compute-1"] = postgres.ComputeResourceRow{ID: "compute-1", OwnerAccountID: "acct-1", State: "creating", IsolationMode: "dedicated_nodepool"}
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
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", OwnerAccountID: "acct-1", ComputeID: "compute-1", StorageID: "storage-1", State: "attaching", MountPath: "/data"}
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
	store.compute["compute-1"] = postgres.ComputeResourceRow{ID: "compute-1", OwnerAccountID: "acct-1", State: "creating", ComputeShapeJSON: `{"cpu":2,"memoryGb":4}`}
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", OwnerAccountID: "acct-1", ComputeID: "compute-1", StorageID: "storage-1", State: "attaching", MountPath: "/data"}
	store.entries["entry-1"] = postgres.WorkspaceEntryRow{ID: "entry-1", OwnerAccountID: "acct-1", WorkspaceID: "ws-1", AttachmentID: "attach-1", State: "creating", Host: "workspace.medopl.cn", Path: "/w/ws-1/"}
	store.workspaces["ws-1"] = postgres.WorkspaceRow{ID: "ws-1", OwnerAccountID: "acct-1", StorageID: "storage-1", ComputeID: "compute-1", AttachmentID: "attach-1", EntryID: "entry-1", OperationID: "op-workspace", State: "provisioning"}
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

func TestApplyWorkspaceOperationFailureLeavesStorageRetained(t *testing.T) {
	store := newMemoryStore()
	store.operations["op-workspace"] = postgres.OperationRow{ID: "op-workspace", ResourceKind: "workspace", ResourceID: "ws-1", State: "accepted"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", OwnerAccountID: "acct-1", SizeGB: 20, State: "creating", Retained: true}
	store.compute["compute-1"] = postgres.ComputeResourceRow{ID: "compute-1", OwnerAccountID: "acct-1", State: "creating"}
	store.attachments["attach-1"] = postgres.StorageAttachmentRow{ID: "attach-1", OwnerAccountID: "acct-1", ComputeID: "compute-1", StorageID: "storage-1", State: "attaching", MountPath: "/data"}
	store.entries["entry-1"] = postgres.WorkspaceEntryRow{ID: "entry-1", OwnerAccountID: "acct-1", WorkspaceID: "ws-1", AttachmentID: "attach-1", State: "creating"}
	store.workspaces["ws-1"] = postgres.WorkspaceRow{ID: "ws-1", StorageID: "storage-1", ComputeID: "compute-1", AttachmentID: "attach-1", EntryID: "entry-1", OperationID: "op-workspace", State: "provisioning"}
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
	store.operations["op-compute"] = postgres.OperationRow{ID: "op-compute", ResourceKind: "compute_destroy", ResourceID: "compute-1", State: "accepted"}
	store.operations["op-storage"] = postgres.OperationRow{ID: "op-storage", ResourceKind: "storage_destroy", ResourceID: "storage-1", State: "accepted"}
	store.compute["compute-1"] = postgres.ComputeResourceRow{ID: "compute-1", State: "running", ProviderRef: "deployment/compute-1"}
	store.storage["storage-1"] = postgres.StorageVolumeRow{ID: "storage-1", State: "available", ProviderRef: "pvc/storage-1", Retained: true}
	runtime := &recordingRuntime{}
	orch := Orchestrator{Store: store, Runtime: runtime}

	if _, err := orch.Apply(context.Background(), "op-compute"); err != nil {
		t.Fatalf("Apply compute destroy: %v", err)
	}
	if _, err := orch.Apply(context.Background(), "op-storage"); err != nil {
		t.Fatalf("Apply storage destroy: %v", err)
	}

	if store.compute["compute-1"].State != "destroyed" || runtime.destroyedComputeID != "compute-1" {
		t.Fatalf("compute destroy failed: row=%+v runtime=%q", store.compute["compute-1"], runtime.destroyedComputeID)
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
	compute     map[string]postgres.ComputeResourceRow
	attachments map[string]postgres.StorageAttachmentRow
	entries     map[string]postgres.WorkspaceEntryRow
	workspaces  map[string]postgres.WorkspaceRow
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		operations:  map[string]postgres.OperationRow{},
		storage:     map[string]postgres.StorageVolumeRow{},
		compute:     map[string]postgres.ComputeResourceRow{},
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

func (s *memoryStore) GetComputeResource(_ context.Context, id string) (postgres.ComputeResourceRow, error) {
	return s.compute[id], nil
}

func (s *memoryStore) UpdateComputeResource(_ context.Context, row postgres.ComputeResourceRow) error {
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
	err                      error
	computeErr               error
	nodePoolID               string
	calls                    []string
	createdStorageID         string
	createdComputeID         string
	attachedID               string
	workspaceEntryID         string
	workspaceEntryServiceRef string
	destroyedComputeID       string
	destroyedStorageID       string
	detachedID               string
}

func (r *recordingRuntime) CreateStorageVolume(_ context.Context, row postgres.StorageVolumeRow) (RuntimeStorageResult, error) {
	if r.err != nil {
		return RuntimeStorageResult{}, r.err
	}
	r.calls = append(r.calls, "storage:"+row.ID)
	r.createdStorageID = row.ID
	return RuntimeStorageResult{ProviderRef: "pvc/" + row.ID}, nil
}

func (r *recordingRuntime) CreateCompute(_ context.Context, row postgres.ComputeResourceRow) (RuntimeComputeResult, error) {
	if r.err != nil {
		return RuntimeComputeResult{}, r.err
	}
	if r.computeErr != nil {
		return RuntimeComputeResult{}, r.computeErr
	}
	r.calls = append(r.calls, "compute:"+row.ID)
	r.createdComputeID = row.ID
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

func (r *recordingRuntime) DestroyCompute(_ context.Context, row postgres.ComputeResourceRow) error {
	if r.err != nil {
		return r.err
	}
	r.destroyedComputeID = row.ID
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
