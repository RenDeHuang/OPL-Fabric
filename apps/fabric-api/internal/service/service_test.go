package service

import (
	"context"
	"slices"
	"testing"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
)

func TestReadinessRequiresProductionFabricKeys(t *testing.T) {
	svc := New(Config{Catalog: catalog.DefaultCatalog(catalog.Config{})})

	readiness := svc.Readiness()

	for _, key := range []string{
		"DATABASE_URL",
		"OPL_OPERATOR_TOKEN",
		"OPL_INGRESS_CLASS",
		"OPL_IMAGE_PULL_SECRET_NAME",
		"TENCENT_TKE_REGION",
		"TENCENT_MUTATION_SECRET_ID",
		"TENCENT_MUTATION_SECRET_KEY",
	} {
		if !slices.Contains(readiness.MissingEnv, key) {
			t.Fatalf("MissingEnv = %v, want %s", readiness.MissingEnv, key)
		}
	}
	if readiness.Ready {
		t.Fatal("readiness should be blocked with missing production keys")
	}
}

func TestReadinessAllowsOptionalCodexSecret(t *testing.T) {
	svc := New(Config{
		Catalog:                    catalog.DefaultCatalog(catalog.Config{}),
		DatabaseURL:                "postgres://example",
		OperatorToken:              "operator",
		KubernetesNamespace:        "oplfabric",
		IngressClass:               "qcloud",
		ImagePullSecretName:        "tcr-pull-secret",
		WorkspaceImage:             "workspace:latest",
		WorkspaceDomain:            "workspace.medopl.cn",
		StorageClass:               "cbs",
		TencentTKERegion:           "ap-guangzhou",
		TencentClusterID:           "cls-example",
		TencentSecretID:            "secret-id",
		TencentSecretKey:           "secret-key",
		TencentTCRRegistry:         "registry.example.com",
		TencentTCRNamespace:        "opl",
		TencentTCRRegion:           "ap-guangzhou",
		TencentCVMSubnetIDs:        "subnet-1",
		TencentCVMSecurityGroupIDs: "sg-1",
	})

	readiness := svc.Readiness()

	if slices.Contains(readiness.MissingEnv, "OPL_CODEX_API_KEY") {
		t.Fatalf("Codex API key should be optional until workspace bootstrap is enabled: %v", readiness.MissingEnv)
	}
}

func TestAcceptWorkspaceDefaultsToWorkspaceExclusiveComputePool(t *testing.T) {
	store := &recordingStore{}
	svc := New(Config{Catalog: catalog.DefaultCatalog(catalog.Config{}), Store: store, WorkspaceDomain: "workspace.medopl.cn", OperatorToken: "operator"})

	_, err := svc.AcceptWorkspace(context.Background(), MutationHeaders{IdempotencyKey: "idem-1", CorrelationID: "corr-1"}, CreateWorkspaceRequest{
		AccountID:       "acct-1",
		RequestedBy:     "console",
		WorkspaceName:   "lab",
		ProductPresetID: "basic",
	})
	if err != nil {
		t.Fatalf("AcceptWorkspace: %v", err)
	}

	compute := store.workspaceReservation.Compute
	if compute.IsolationMode != "workspace_exclusive_cvm" {
		t.Fatalf("IsolationMode = %q", compute.IsolationMode)
	}
	if compute.CapacityPoolID != "tencent-cpu-compute-pool" {
		t.Fatalf("CapacityPoolID = %q", compute.CapacityPoolID)
	}
	if compute.ComputeShapeJSON != `{"cpu":2,"gpu":0,"memoryGb":4}` {
		t.Fatalf("ComputeShapeJSON = %s", compute.ComputeShapeJSON)
	}
}

func TestAcceptComputeAllocationDefaultsToWorkspaceExclusiveComputePool(t *testing.T) {
	store := &recordingStore{}
	svc := New(Config{Catalog: catalog.DefaultCatalog(catalog.Config{}), Store: store, OperatorToken: "operator"})

	_, err := svc.AcceptComputeAllocation(context.Background(), MutationHeaders{IdempotencyKey: "idem-1", CorrelationID: "corr-1"}, CreateComputeAllocationRequest{
		AccountID:       "acct-1",
		RequestedBy:     "console",
		ProductPresetID: "basic",
	})
	if err != nil {
		t.Fatalf("AcceptComputeAllocation: %v", err)
	}

	compute := store.compute
	if compute.IsolationMode != "workspace_exclusive_cvm" {
		t.Fatalf("IsolationMode = %q", compute.IsolationMode)
	}
	if compute.CapacityPoolID != "tencent-cpu-compute-pool" {
		t.Fatalf("CapacityPoolID = %q", compute.CapacityPoolID)
	}
	if compute.ComputeShapeJSON != `{"cpu":2,"gpu":0,"memoryGb":4}` {
		t.Fatalf("ComputeShapeJSON = %s", compute.ComputeShapeJSON)
	}
}

type recordingStore struct {
	operation            postgres.OperationRow
	storage              postgres.StorageVolumeRow
	compute              postgres.ComputeAllocationRow
	attachment           postgres.StorageAttachmentRow
	entry                postgres.WorkspaceEntryRow
	workspace            postgres.WorkspaceRow
	workspaceReservation postgres.WorkspaceReservation
}

func (s *recordingStore) CreateOperation(_ context.Context, row postgres.OperationRow) error {
	s.operation = row
	return nil
}

func (s *recordingStore) GetOperation(_ context.Context, _ string) (postgres.OperationRow, error) {
	return s.operation, nil
}

func (s *recordingStore) CreateStorageVolume(_ context.Context, row postgres.StorageVolumeRow) error {
	s.storage = row
	return nil
}

func (s *recordingStore) CreateComputeAllocation(_ context.Context, row postgres.ComputeAllocationRow) error {
	s.compute = row
	return nil
}

func (s *recordingStore) CreateStorageAttachment(_ context.Context, row postgres.StorageAttachmentRow) error {
	s.attachment = row
	return nil
}

func (s *recordingStore) CreateWorkspaceEntry(_ context.Context, row postgres.WorkspaceEntryRow) error {
	s.entry = row
	return nil
}

func (s *recordingStore) CreateWorkspace(_ context.Context, row postgres.WorkspaceRow) error {
	s.workspace = row
	return nil
}

func (s *recordingStore) CreateWorkspaceReservation(_ context.Context, row postgres.WorkspaceReservation) error {
	s.workspaceReservation = row
	return nil
}

func (s *recordingStore) GetWorkspace(_ context.Context, _ string) (postgres.WorkspaceRow, error) {
	return s.workspace, nil
}

func (s *recordingStore) GetStorageVolume(_ context.Context, _ string) (postgres.StorageVolumeRow, error) {
	return s.storage, nil
}

func (s *recordingStore) GetComputeAllocation(_ context.Context, _ string) (postgres.ComputeAllocationRow, error) {
	return s.compute, nil
}

func (s *recordingStore) GetStorageAttachment(_ context.Context, _ string) (postgres.StorageAttachmentRow, error) {
	return s.attachment, nil
}

func (s *recordingStore) GetWorkspaceEntry(_ context.Context, _ string) (postgres.WorkspaceEntryRow, error) {
	return s.entry, nil
}
