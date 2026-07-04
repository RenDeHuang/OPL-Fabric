package staging

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/orchestrator"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/worker"
)

func TestFakeWorkspaceChainRunsStorageComputeAttachEntry(t *testing.T) {
	ctx := context.Background()
	store := newChainStore()
	svc := service.New(service.Config{
		Catalog:         catalog.DefaultCatalog(catalog.Config{WorkspaceImage: "workspace:staging", WorkspaceDomain: "workspace.medopl.cn", StorageClass: "cbs"}),
		DatabaseURL:     "postgres://fake",
		OperatorToken:   "operator-token",
		WorkspaceImage:  "workspace:staging",
		WorkspaceDomain: "workspace.medopl.cn",
		StorageClass:    "cbs",
		Store:           store,
	})
	receipt, err := svc.AcceptWorkspace(ctx, service.MutationHeaders{
		IdempotencyKey: "workspace-chain-1",
		CorrelationID:  "corr-workspace-chain-1",
	}, service.CreateWorkspaceRequest{
		AccountID:            "acct-1",
		RequestedBy:          "user-1",
		WorkspaceName:        "Lab",
		ProductPresetID:      "basic",
		ComputeShape:         map[string]any{"cpu": 2, "memoryGb": 4},
		ProviderInstanceType: "SA5.LARGE8",
		CapacityPoolID:       "shared-basic",
		IsolationMode:        "shared_pool",
		Storage: struct {
			SizeGB int `json:"sizeGb"`
		}{SizeGB: 20},
	})
	if err != nil {
		t.Fatalf("AcceptWorkspace: %v", err)
	}

	runtime := &recordingChainRuntime{}
	w := worker.Worker{
		Store:        store,
		Orchestrator: orchestrator.Orchestrator{Store: store, Runtime: runtime},
		Owner:        "staging-test-worker",
		BatchSize:    10,
		LeaseTTL:     time.Minute,
	}
	if err := w.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	workspace, err := svc.Workspace(ctx, receipt.ResourceID)
	if err != nil {
		t.Fatalf("Workspace: %v", err)
	}
	if workspace.State != "running" || workspace.Storage.State != "available" || workspace.Compute.State != "running" || workspace.Attachment.State != "attached" || workspace.Entry.State != "ready" {
		t.Fatalf("workspace chain did not complete: %+v", workspace)
	}
	if store.operations[receipt.OperationID].State != "succeeded" {
		t.Fatalf("operation state = %q", store.operations[receipt.OperationID].State)
	}
	wantCalls := []string{"storage", "compute", "attach", "entry"}
	if !reflect.DeepEqual(runtime.calls, wantCalls) {
		t.Fatalf("runtime calls = %v, want %v", runtime.calls, wantCalls)
	}
	if store.attachments[workspace.Attachment.ID].ProviderRef == "" {
		t.Fatal("attachment provider ref was not populated")
	}
	if store.entries[workspace.Entry.ID].ServiceRef == "" {
		t.Fatal("workspace entry service ref was not populated")
	}
}

func TestLiveWorkspaceChainE2EIsGated(t *testing.T) {
	if os.Getenv("OPL_STAGING_E2E_ALLOW_LIVE") != "true" || os.Getenv("OPL_FABRIC_WORKER_ENABLED") != "true" {
		t.Skip("live staging e2e requires OPL_STAGING_E2E_ALLOW_LIVE=true and OPL_FABRIC_WORKER_ENABLED=true")
	}
	result := EvaluateGate(Config{
		DatabaseURL:                os.Getenv("DATABASE_URL"),
		OperatorToken:              os.Getenv("OPL_OPERATOR_TOKEN"),
		KubeconfigRef:              os.Getenv("TENCENT_DEPLOY_KUBECONFIG_REF"),
		Namespace:                  os.Getenv("OPL_K8S_NAMESPACE"),
		StorageClass:               os.Getenv("OPL_WORKSPACE_STORAGE_CLASS"),
		IngressClass:               os.Getenv("OPL_INGRESS_CLASS"),
		WorkspaceDomain:            os.Getenv("OPL_WORKSPACE_DOMAIN"),
		WorkspaceImage:             os.Getenv("OPL_WORKSPACE_IMAGE"),
		ImagePullSecretName:        os.Getenv("OPL_IMAGE_PULL_SECRET_NAME"),
		TencentClusterID:           os.Getenv("TENCENT_DEPLOY_CLUSTER_ID"),
		TencentRegion:              os.Getenv("TENCENT_TKE_REGION"),
		TencentSecretID:            os.Getenv("TENCENT_MUTATION_SECRET_ID"),
		TencentSecretKey:           os.Getenv("TENCENT_MUTATION_SECRET_KEY"),
		TencentTCRRegistry:         os.Getenv("TENCENT_TCR_REGISTRY"),
		TencentTCRNamespace:        os.Getenv("TENCENT_TCR_NAMESPACE"),
		TencentTCRRegion:           os.Getenv("TENCENT_TCR_REGION"),
		TencentCVMSubnetIDs:        os.Getenv("TENCENT_CVM_SUBNET_ID"),
		TencentCVMSecurityGroupIDs: os.Getenv("TENCENT_CVM_SECURITY_GROUP_IDS"),
		TencentCVMSystemDiskType:   os.Getenv("TENCENT_CVM_SYSTEM_DISK_TYPE"),
		TencentCVMSystemDiskSizeGB: os.Getenv("TENCENT_CVM_SYSTEM_DISK_SIZE_GB"),
		AllowNodePoolMutation:      envBool("OPL_TKE_ALLOW_NODEPOOL_MUTATION"),
		AllowStagingE2E:            envBool("OPL_STAGING_E2E_ALLOW_LIVE"),
		WorkerEnabled:              envBool("OPL_FABRIC_WORKER_ENABLED"),
	})
	if !result.Ready || result.Mode != "ready_for_live" {
		t.Fatalf("live staging e2e gate blocked: %+v", result)
	}
	t.Skip("live staging e2e execution is reserved for the staging pipeline to avoid accidental local cloud mutation")
}

type chainStore struct {
	operations  map[string]postgres.OperationRow
	storages    map[string]postgres.StorageVolumeRow
	computes    map[string]postgres.ComputeResourceRow
	attachments map[string]postgres.StorageAttachmentRow
	entries     map[string]postgres.WorkspaceEntryRow
	workspaces  map[string]postgres.WorkspaceRow
}

func newChainStore() *chainStore {
	return &chainStore{
		operations:  map[string]postgres.OperationRow{},
		storages:    map[string]postgres.StorageVolumeRow{},
		computes:    map[string]postgres.ComputeResourceRow{},
		attachments: map[string]postgres.StorageAttachmentRow{},
		entries:     map[string]postgres.WorkspaceEntryRow{},
		workspaces:  map[string]postgres.WorkspaceRow{},
	}
}

func (s *chainStore) CreateOperation(_ context.Context, row postgres.OperationRow) error {
	s.operations[row.ID] = row
	return nil
}

func (s *chainStore) GetOperation(_ context.Context, id string) (postgres.OperationRow, error) {
	row, ok := s.operations[id]
	if !ok {
		return postgres.OperationRow{}, errors.New("operation_not_found")
	}
	return row, nil
}

func (s *chainStore) UpdateOperationState(_ context.Context, id, state string) error {
	row, ok := s.operations[id]
	if !ok {
		return errors.New("operation_not_found")
	}
	row.State = state
	s.operations[id] = row
	return nil
}

func (s *chainStore) ListAcceptedOperations(_ context.Context, limit int) ([]postgres.OperationRow, error) {
	rows := []postgres.OperationRow{}
	for _, row := range s.operations {
		if row.State != "accepted" {
			continue
		}
		rows = append(rows, row)
		if limit > 0 && len(rows) >= limit {
			break
		}
	}
	return rows, nil
}

func (s *chainStore) LeaseOperation(_ context.Context, id, owner string, _ time.Duration) (bool, error) {
	row, ok := s.operations[id]
	if !ok || row.State != "accepted" {
		return false, nil
	}
	row.LeaseOwner = owner
	s.operations[id] = row
	return true, nil
}

func (s *chainStore) RecordOperationFailure(_ context.Context, id string, cause error) error {
	row, ok := s.operations[id]
	if !ok {
		return errors.New("operation_not_found")
	}
	row.State = "failed"
	if cause != nil {
		row.LastError = cause.Error()
	}
	s.operations[id] = row
	return nil
}

func (s *chainStore) CreateStorageVolume(_ context.Context, row postgres.StorageVolumeRow) error {
	s.storages[row.ID] = row
	return nil
}

func (s *chainStore) GetStorageVolume(_ context.Context, id string) (postgres.StorageVolumeRow, error) {
	row, ok := s.storages[id]
	if !ok {
		return postgres.StorageVolumeRow{}, errors.New("storage_not_found")
	}
	return row, nil
}

func (s *chainStore) UpdateStorageVolume(_ context.Context, row postgres.StorageVolumeRow) error {
	s.storages[row.ID] = row
	return nil
}

func (s *chainStore) CreateComputeResource(_ context.Context, row postgres.ComputeResourceRow) error {
	s.computes[row.ID] = row
	return nil
}

func (s *chainStore) GetComputeResource(_ context.Context, id string) (postgres.ComputeResourceRow, error) {
	row, ok := s.computes[id]
	if !ok {
		return postgres.ComputeResourceRow{}, errors.New("compute_not_found")
	}
	return row, nil
}

func (s *chainStore) UpdateComputeResource(_ context.Context, row postgres.ComputeResourceRow) error {
	s.computes[row.ID] = row
	return nil
}

func (s *chainStore) CreateStorageAttachment(_ context.Context, row postgres.StorageAttachmentRow) error {
	s.attachments[row.ID] = row
	return nil
}

func (s *chainStore) GetStorageAttachment(_ context.Context, id string) (postgres.StorageAttachmentRow, error) {
	row, ok := s.attachments[id]
	if !ok {
		return postgres.StorageAttachmentRow{}, errors.New("attachment_not_found")
	}
	return row, nil
}

func (s *chainStore) UpdateStorageAttachment(_ context.Context, row postgres.StorageAttachmentRow) error {
	s.attachments[row.ID] = row
	return nil
}

func (s *chainStore) CreateWorkspaceEntry(_ context.Context, row postgres.WorkspaceEntryRow) error {
	s.entries[row.ID] = row
	return nil
}

func (s *chainStore) GetWorkspaceEntry(_ context.Context, id string) (postgres.WorkspaceEntryRow, error) {
	row, ok := s.entries[id]
	if !ok {
		return postgres.WorkspaceEntryRow{}, errors.New("entry_not_found")
	}
	return row, nil
}

func (s *chainStore) UpdateWorkspaceEntry(_ context.Context, row postgres.WorkspaceEntryRow) error {
	s.entries[row.ID] = row
	return nil
}

func (s *chainStore) CreateWorkspace(_ context.Context, row postgres.WorkspaceRow) error {
	s.workspaces[row.ID] = row
	return nil
}

func (s *chainStore) CreateWorkspaceReservation(ctx context.Context, reservation postgres.WorkspaceReservation) error {
	if err := s.CreateOperation(ctx, reservation.Operation); err != nil {
		return err
	}
	if err := s.CreateStorageVolume(ctx, reservation.Storage); err != nil {
		return err
	}
	if err := s.CreateComputeResource(ctx, reservation.Compute); err != nil {
		return err
	}
	if err := s.CreateStorageAttachment(ctx, reservation.Attachment); err != nil {
		return err
	}
	if err := s.CreateWorkspaceEntry(ctx, reservation.Entry); err != nil {
		return err
	}
	return s.CreateWorkspace(ctx, reservation.Workspace)
}

func (s *chainStore) GetWorkspace(_ context.Context, id string) (postgres.WorkspaceRow, error) {
	row, ok := s.workspaces[id]
	if !ok {
		return postgres.WorkspaceRow{}, errors.New("workspace_not_found")
	}
	return row, nil
}

func (s *chainStore) UpdateWorkspace(_ context.Context, row postgres.WorkspaceRow) error {
	s.workspaces[row.ID] = row
	return nil
}

type recordingChainRuntime struct {
	calls []string
}

func (r *recordingChainRuntime) CreateStorageVolume(_ context.Context, row postgres.StorageVolumeRow) (orchestrator.RuntimeStorageResult, error) {
	r.calls = append(r.calls, "storage")
	return orchestrator.RuntimeStorageResult{ProviderRef: "pvc/" + row.ID}, nil
}

func (r *recordingChainRuntime) CreateCompute(_ context.Context, row postgres.ComputeResourceRow) (orchestrator.RuntimeComputeResult, error) {
	r.calls = append(r.calls, "compute")
	return orchestrator.RuntimeComputeResult{ProviderRef: "deployment/" + row.ID, RuntimeRef: "service/" + row.ID}, nil
}

func (r *recordingChainRuntime) AttachStorage(_ context.Context, row postgres.StorageAttachmentRow) (orchestrator.RuntimeAttachmentResult, error) {
	r.calls = append(r.calls, "attach")
	return orchestrator.RuntimeAttachmentResult{ProviderRef: row.ProviderRef}, nil
}

func (r *recordingChainRuntime) CreateWorkspaceEntry(_ context.Context, row postgres.WorkspaceEntryRow) error {
	r.calls = append(r.calls, "entry")
	if row.ServiceRef == "" {
		return errors.New("entry_service_ref_required")
	}
	return nil
}

func (r *recordingChainRuntime) DestroyCompute(context.Context, postgres.ComputeResourceRow) error {
	return nil
}

func (r *recordingChainRuntime) DestroyStorage(context.Context, postgres.StorageVolumeRow) error {
	return nil
}

func (r *recordingChainRuntime) DetachStorage(context.Context, postgres.StorageAttachmentRow) error {
	return nil
}

func envBool(key string) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	return err == nil && value
}
