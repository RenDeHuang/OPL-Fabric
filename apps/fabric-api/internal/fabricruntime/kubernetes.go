package fabricruntime

import (
	"context"

	fabrick8s "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/k8s"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/orchestrator"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
)

type KubernetesRuntime struct {
	Provider fabrick8s.Provider
}

func (r KubernetesRuntime) CreateStorageVolume(ctx context.Context, row postgres.StorageVolumeRow) (orchestrator.RuntimeStorageResult, error) {
	result, err := r.Provider.CreateStorageVolume(ctx, fabrick8s.CreateStorageVolumeInput{ID: row.ID, SizeGB: row.SizeGB})
	return orchestrator.RuntimeStorageResult{ProviderRef: result.ProviderRef}, err
}

func (r KubernetesRuntime) CreateCompute(ctx context.Context, row postgres.ComputeResourceRow) (orchestrator.RuntimeComputeResult, error) {
	result, err := r.Provider.CreateCompute(ctx, fabrick8s.CreateComputeInput{
		ID:                   row.ID,
		ProductPresetID:      row.ProductPresetID,
		ComputeShapeJSON:     row.ComputeShapeJSON,
		ProviderInstanceType: row.ProviderInstanceType,
		CapacityPoolID:       row.CapacityPoolID,
		IsolationMode:        row.IsolationMode,
		NodePoolID:           row.NodePoolID,
		RuntimeRef:           row.RuntimeRef,
	})
	return orchestrator.RuntimeComputeResult{ProviderRef: result.ProviderRef, RuntimeRef: result.ServiceRef}, err
}

func (r KubernetesRuntime) AttachStorage(ctx context.Context, row postgres.StorageAttachmentRow) (orchestrator.RuntimeAttachmentResult, error) {
	computeRef, storageRef := splitProviderRefs(row.ProviderRef, row.ComputeID, row.StorageID)
	result, err := r.Provider.AttachStorage(ctx, fabrick8s.AttachStorageInput{
		ID:         row.ID,
		ComputeRef: computeRef,
		StorageRef: storageRef,
		MountPath:  row.MountPath,
	})
	return orchestrator.RuntimeAttachmentResult{ProviderRef: result.ProviderRef}, err
}

func (r KubernetesRuntime) CreateWorkspaceEntry(ctx context.Context, row postgres.WorkspaceEntryRow) error {
	return r.Provider.CreateWorkspaceEntry(ctx, fabrick8s.CreateWorkspaceEntryInput{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Host:        row.Host,
		Path:        row.Path,
		ServiceRef:  "service/" + row.WorkspaceID,
	})
}

func (r KubernetesRuntime) DestroyCompute(ctx context.Context, row postgres.ComputeResourceRow) error {
	return r.Provider.DestroyCompute(ctx, fabrick8s.DestroyComputeInput{ProviderRef: row.ProviderRef, RuntimeRef: row.RuntimeRef})
}

func (r KubernetesRuntime) DestroyStorage(ctx context.Context, row postgres.StorageVolumeRow) error {
	return r.Provider.DestroyStorage(ctx, fabrick8s.DestroyStorageInput{ProviderRef: row.ProviderRef})
}

func (r KubernetesRuntime) DetachStorage(ctx context.Context, row postgres.StorageAttachmentRow) error {
	return r.Provider.DetachStorage(ctx, fabrick8s.DetachStorageInput{ProviderRef: row.ProviderRef})
}

func splitProviderRefs(providerRef, computeID, storageID string) (computeRef, storageRef string) {
	for i, value := range splitPair(providerRef) {
		if i == 0 {
			computeRef = value
		}
		if i == 1 {
			storageRef = value
		}
	}
	if computeRef == "" {
		computeRef = "deployment/" + computeID
	}
	if storageRef == "" {
		storageRef = "pvc/" + storageID
	}
	return computeRef, storageRef
}

func splitPair(value string) []string {
	if value == "" {
		return nil
	}
	for i, char := range value {
		if char == ':' {
			return []string{value[:i], value[i+1:]}
		}
	}
	return []string{value}
}
