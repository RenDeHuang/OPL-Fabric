package domain

import "errors"

var (
	ErrStorageDestroyRequiresConfirmation = errors.New("storage_destroy_requires_confirmation")
	ErrRequestedByRequired                = errors.New("requested_by_required")
	ErrStorageIDRequired                  = errors.New("storage_id_required")
	ErrStorageAlreadyDestroyed            = errors.New("storage_already_destroyed")
)

func CanDestroyStorage(storage StorageVolume, req DestroyStorageRequest) error {
	if req.RequestedBy == "" {
		return ErrRequestedByRequired
	}
	if storage.ID == "" {
		return ErrStorageIDRequired
	}
	if storage.State == StorageDestroyed || storage.State == StorageDestroying {
		return ErrStorageAlreadyDestroyed
	}
	if !req.Confirm && req.HumanGateID == "" {
		return ErrStorageDestroyRequiresConfirmation
	}
	return nil
}

func DestroyCompute(compute ComputeResource, storage StorageVolume) (DestroyResult, error) {
	compute.State = ComputeDestroying
	return DestroyResult{Compute: compute, Storage: storage}, nil
}
