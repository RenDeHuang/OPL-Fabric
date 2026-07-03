package domain

import "errors"

var (
	ErrStorageDestroyRequiresConfirmation = errors.New("storage_destroy_requires_confirmation")
	ErrRequestedByRequired                = errors.New("requested_by_required")
)

func CanDestroyStorage(storage StorageVolume, req DestroyStorageRequest) error {
	if req.RequestedBy == "" {
		return ErrRequestedByRequired
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
