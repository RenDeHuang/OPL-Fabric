package orchestrator

import (
	"context"
	"errors"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
)

var ErrUnsupportedOperationKind = errors.New("unsupported_operation_kind")

type Store interface {
	GetOperation(context.Context, string) (postgres.OperationRow, error)
	UpdateOperationState(context.Context, string, string) error
	GetStorageVolume(context.Context, string) (postgres.StorageVolumeRow, error)
	UpdateStorageVolume(context.Context, postgres.StorageVolumeRow) error
	GetComputeAllocation(context.Context, string) (postgres.ComputeAllocationRow, error)
	UpdateComputeAllocation(context.Context, postgres.ComputeAllocationRow) error
	GetStorageAttachment(context.Context, string) (postgres.StorageAttachmentRow, error)
	UpdateStorageAttachment(context.Context, postgres.StorageAttachmentRow) error
	GetWorkspaceEntry(context.Context, string) (postgres.WorkspaceEntryRow, error)
	UpdateWorkspaceEntry(context.Context, postgres.WorkspaceEntryRow) error
	GetWorkspace(context.Context, string) (postgres.WorkspaceRow, error)
	UpdateWorkspace(context.Context, postgres.WorkspaceRow) error
}

type Runtime interface {
	CreateStorageVolume(context.Context, postgres.StorageVolumeRow) (RuntimeStorageResult, error)
	CreateCompute(context.Context, postgres.ComputeAllocationRow) (RuntimeComputeResult, error)
	AttachStorage(context.Context, postgres.StorageAttachmentRow) (RuntimeAttachmentResult, error)
	CreateWorkspaceEntry(context.Context, postgres.WorkspaceEntryRow) error
	DestroyCompute(context.Context, postgres.ComputeAllocationRow) error
	DestroyStorage(context.Context, postgres.StorageVolumeRow) error
	DetachStorage(context.Context, postgres.StorageAttachmentRow) error
}

type RuntimeStorageResult struct {
	ProviderRef string
}

type RuntimeComputeResult struct {
	ProviderRef string
	RuntimeRef  string
	NodePoolID  string
}

type RuntimeAttachmentResult struct {
	ProviderRef string
}

type Receipt struct {
	OperationID  string
	State        string
	ResourceKind string
	ResourceID   string
}

type Orchestrator struct {
	Store   Store
	Runtime Runtime
}

func (o Orchestrator) Apply(ctx context.Context, operationID string) (Receipt, error) {
	op, err := o.Store.GetOperation(ctx, operationID)
	if err != nil {
		return Receipt{}, err
	}
	if err := o.Store.UpdateOperationState(ctx, op.ID, "applying"); err != nil {
		return Receipt{}, err
	}
	if err := o.apply(ctx, op); err != nil {
		_ = o.Store.UpdateOperationState(ctx, op.ID, "failed")
		return Receipt{}, err
	}
	if err := o.Store.UpdateOperationState(ctx, op.ID, "succeeded"); err != nil {
		return Receipt{}, err
	}
	return Receipt{OperationID: op.ID, State: "succeeded", ResourceKind: op.ResourceKind, ResourceID: op.ResourceID}, nil
}

func (o Orchestrator) apply(ctx context.Context, op postgres.OperationRow) error {
	switch op.ResourceKind {
	case "storage_volume":
		return o.applyStorageVolume(ctx, op.ResourceID)
	case "compute_allocation":
		return o.applyComputeAllocation(ctx, op.ResourceID)
	case "storage_attachment":
		return o.applyStorageAttachment(ctx, op.ResourceID)
	case "workspace_entry":
		return o.applyWorkspaceEntry(ctx, op.ResourceID)
	case "workspace":
		return o.applyWorkspace(ctx, op.ResourceID)
	case "compute_allocation_destroy":
		return o.applyComputeAllocationDestroy(ctx, op.ResourceID)
	case "storage_destroy":
		return o.applyStorageDestroy(ctx, op.ResourceID)
	case "attachment_detach":
		return o.applyAttachmentDetach(ctx, op.ResourceID)
	default:
		return ErrUnsupportedOperationKind
	}
}

func (o Orchestrator) applyStorageVolume(ctx context.Context, id string) error {
	row, err := o.Store.GetStorageVolume(ctx, id)
	if err != nil {
		return err
	}
	result, err := o.Runtime.CreateStorageVolume(ctx, row)
	if err != nil {
		return err
	}
	row.ProviderRef = result.ProviderRef
	row.State = "available"
	return o.Store.UpdateStorageVolume(ctx, row)
}

func (o Orchestrator) applyComputeAllocation(ctx context.Context, id string) error {
	row, err := o.Store.GetComputeAllocation(ctx, id)
	if err != nil {
		return err
	}
	result, err := o.Runtime.CreateCompute(ctx, row)
	if err != nil {
		return err
	}
	row.ProviderRef = result.ProviderRef
	row.RuntimeRef = result.RuntimeRef
	row.NodePoolID = result.NodePoolID
	row.State = "running"
	return o.Store.UpdateComputeAllocation(ctx, row)
}

func (o Orchestrator) applyStorageAttachment(ctx context.Context, id string) error {
	row, err := o.Store.GetStorageAttachment(ctx, id)
	if err != nil {
		return err
	}
	result, err := o.Runtime.AttachStorage(ctx, row)
	if err != nil {
		return err
	}
	row.ProviderRef = result.ProviderRef
	row.State = "attached"
	return o.Store.UpdateStorageAttachment(ctx, row)
}

func (o Orchestrator) applyWorkspaceEntry(ctx context.Context, id string) error {
	row, err := o.Store.GetWorkspaceEntry(ctx, id)
	if err != nil {
		return err
	}
	if err := o.Runtime.CreateWorkspaceEntry(ctx, row); err != nil {
		return err
	}
	row.State = "ready"
	return o.Store.UpdateWorkspaceEntry(ctx, row)
}

func (o Orchestrator) applyWorkspace(ctx context.Context, id string) error {
	workspace, err := o.Store.GetWorkspace(ctx, id)
	if err != nil {
		return err
	}
	workspace.State = "provisioning"
	if err := o.Store.UpdateWorkspace(ctx, workspace); err != nil {
		return err
	}
	if err := o.applyStorageVolume(ctx, workspace.StorageID); err != nil {
		return err
	}
	if err := o.applyComputeAllocation(ctx, workspace.ComputeAllocationID); err != nil {
		return err
	}
	compute, err := o.Store.GetComputeAllocation(ctx, workspace.ComputeAllocationID)
	if err != nil {
		return err
	}
	storage, err := o.Store.GetStorageVolume(ctx, workspace.StorageID)
	if err != nil {
		return err
	}
	attachment, err := o.Store.GetStorageAttachment(ctx, workspace.AttachmentID)
	if err != nil {
		return err
	}
	attachment.ProviderRef = compute.ProviderRef + ":" + storage.ProviderRef
	if err := o.Store.UpdateStorageAttachment(ctx, attachment); err != nil {
		return err
	}
	if err := o.applyStorageAttachment(ctx, workspace.AttachmentID); err != nil {
		return err
	}
	entry, err := o.Store.GetWorkspaceEntry(ctx, workspace.EntryID)
	if err != nil {
		return err
	}
	entry.ServiceRef = compute.RuntimeRef
	if err := o.Store.UpdateWorkspaceEntry(ctx, entry); err != nil {
		return err
	}
	if err := o.applyWorkspaceEntry(ctx, workspace.EntryID); err != nil {
		return err
	}
	workspace.State = "running"
	return o.Store.UpdateWorkspace(ctx, workspace)
}

func (o Orchestrator) applyComputeAllocationDestroy(ctx context.Context, id string) error {
	row, err := o.Store.GetComputeAllocation(ctx, id)
	if err != nil {
		return err
	}
	if err := o.Runtime.DestroyCompute(ctx, row); err != nil {
		return err
	}
	row.State = "destroyed"
	row.ProviderRef = ""
	row.RuntimeRef = ""
	row.NodePoolID = ""
	return o.Store.UpdateComputeAllocation(ctx, row)
}

func (o Orchestrator) applyStorageDestroy(ctx context.Context, id string) error {
	row, err := o.Store.GetStorageVolume(ctx, id)
	if err != nil {
		return err
	}
	if row.Retained {
		return nil
	}
	if err := o.Runtime.DestroyStorage(ctx, row); err != nil {
		return err
	}
	row.State = "destroyed"
	row.ProviderRef = ""
	return o.Store.UpdateStorageVolume(ctx, row)
}

func (o Orchestrator) applyAttachmentDetach(ctx context.Context, id string) error {
	row, err := o.Store.GetStorageAttachment(ctx, id)
	if err != nil {
		return err
	}
	if err := o.Runtime.DetachStorage(ctx, row); err != nil {
		return err
	}
	row.State = "detached"
	row.ProviderRef = ""
	return o.Store.UpdateStorageAttachment(ctx, row)
}
