package domain

import "testing"

func TestDestroyStorageRequiresConfirmation(t *testing.T) {
	resource := StorageVolume{ID: "storage-1", State: StorageAvailable}

	err := CanDestroyStorage(resource, DestroyStorageRequest{})
	if err == nil {
		t.Fatal("expected destroy storage without confirmation to fail")
	}

	err = CanDestroyStorage(resource, DestroyStorageRequest{Confirm: true, RequestedBy: "operator"})
	if err != nil {
		t.Fatalf("expected confirmed destroy storage to pass: %v", err)
	}
}

func TestDestroyComputeDoesNotDestroyStorage(t *testing.T) {
	compute := ComputeResource{ID: "compute-1", State: ComputeRunning}
	storage := StorageVolume{ID: "storage-1", State: StorageAttached}

	next, err := DestroyCompute(compute, storage)
	if err != nil {
		t.Fatalf("destroy compute failed: %v", err)
	}
	if next.Compute.State != ComputeDestroying {
		t.Fatalf("compute state = %s", next.Compute.State)
	}
	if next.Storage.State != StorageAttached {
		t.Fatalf("storage state changed to %s", next.Storage.State)
	}
}
