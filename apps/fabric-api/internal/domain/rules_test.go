package domain

import "testing"

func TestDestroyStorageRequiresConfirmation(t *testing.T) {
	resource := StorageVolume{ID: "storage-1", State: StorageAvailable}

	tests := []struct {
		name    string
		storage StorageVolume
		req     DestroyStorageRequest
		wantErr error
	}{
		{
			name:    "missing_requested_by",
			storage: resource,
			req:     DestroyStorageRequest{Confirm: true},
			wantErr: ErrRequestedByRequired,
		},
		{
			name:    "missing_confirmation",
			storage: resource,
			req:     DestroyStorageRequest{RequestedBy: "operator"},
			wantErr: ErrStorageDestroyRequiresConfirmation,
		},
		{
			name:    "missing_storage_id",
			storage: StorageVolume{State: StorageAvailable},
			req:     DestroyStorageRequest{Confirm: true, RequestedBy: "operator"},
			wantErr: ErrStorageIDRequired,
		},
		{
			name:    "already_destroyed",
			storage: StorageVolume{ID: "storage-1", State: StorageDestroyed},
			req:     DestroyStorageRequest{Confirm: true, RequestedBy: "operator"},
			wantErr: ErrStorageAlreadyDestroyed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanDestroyStorage(tt.storage, tt.req)
			if err != tt.wantErr {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	err := CanDestroyStorage(resource, DestroyStorageRequest{Confirm: true, RequestedBy: "operator"})
	if err != nil {
		t.Fatalf("expected confirmed destroy storage to pass: %v", err)
	}
}

func TestDestroyComputeDoesNotDestroyStorage(t *testing.T) {
	compute := ComputeAllocation{ID: "compute-1", State: ComputeRunning}
	storage := StorageVolume{ID: "storage-1", State: StorageAttached}

	next, err := DestroyCompute(compute, storage)
	if err != nil {
		t.Fatalf("destroy compute failed: %v", err)
	}
	if next.Compute.State != ComputeAllocationDestroying {
		t.Fatalf("compute state = %s", next.Compute.State)
	}
	if next.Storage.State != StorageAttached {
		t.Fatalf("storage state changed to %s", next.Storage.State)
	}
}
