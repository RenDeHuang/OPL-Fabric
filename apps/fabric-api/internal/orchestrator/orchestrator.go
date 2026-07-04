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
	GetComputeResource(context.Context, string) (postgres.ComputeResourceRow, error)
	UpdateComputeResource(context.Context, postgres.ComputeResourceRow) error
	GetStorageAttachment(context.Context, string) (postgres.StorageAttachmentRow, error)
	UpdateStorageAttachment(context.Context, postgres.StorageAttachmentRow) error
	GetWorkspaceEntry(context.Context, string) (postgres.WorkspaceEntryRow, error)
	UpdateWorkspaceEntry(context.Context, postgres.WorkspaceEntryRow) error
}

type Runtime interface {
	CreateStorageVolume(context.Context, postgres.StorageVolumeRow) (RuntimeStorageResult, error)
	CreateCompute(context.Context, postgres.ComputeResourceRow) (RuntimeComputeResult, error)
	AttachStorage(context.Context, postgres.StorageAttachmentRow) (RuntimeAttachmentResult, error)
	CreateWorkspaceEntry(context.Context, postgres.WorkspaceEntryRow) error
	DestroyCompute(context.Context, postgres.ComputeResourceRow) error
	DestroyStorage(context.Context, postgres.StorageVolumeRow) error
	DetachStorage(context.Context, postgres.StorageAttachmentRow) error
}

type RuntimeStorageResult struct {
	ProviderRef string
}

type RuntimeComputeResult struct {
	ProviderRef string
	RuntimeRef  string
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
	case "compute_resource":
		return o.applyComputeResource(ctx, op.ResourceID)
	case "storage_attachment":
		return o.applyStorageAttachment(ctx, op.ResourceID)
	case "workspace_entry":
		return o.applyWorkspaceEntry(ctx, op.ResourceID)
	case "compute_destroy":
		return o.applyComputeDestroy(ctx, op.ResourceID)
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

func (o Orchestrator) applyComputeResource(ctx context.Context, id string) error {
	row, err := o.Store.GetComputeResource(ctx, id)
	if err != nil {
		return err
	}
	result, err := o.Runtime.CreateCompute(ctx, row)
	if err != nil {
		return err
	}
	row.ProviderRef = result.ProviderRef
	row.RuntimeRef = result.RuntimeRef
	row.State = "running"
	return o.Store.UpdateComputeResource(ctx, row)
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

func (o Orchestrator) applyComputeDestroy(ctx context.Context, id string) error {
	row, err := o.Store.GetComputeResource(ctx, id)
	if err != nil {
		return err
	}
	if err := o.Runtime.DestroyCompute(ctx, row); err != nil {
		return err
	}
	row.State = "destroyed"
	row.ProviderRef = ""
	row.RuntimeRef = ""
	return o.Store.UpdateComputeResource(ctx, row)
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
