package domain

type ComputeState string
type StorageState string
type OperationState string

const (
	ComputeCreating        ComputeState = "creating"
	ComputeRunning         ComputeState = "running"
	ComputeStopping        ComputeState = "stopping"
	ComputeStopped         ComputeState = "stopped"
	ComputeRestarting      ComputeState = "restarting"
	ComputeDestroying      ComputeState = "destroying"
	ComputeDestroyed       ComputeState = "destroyed"
	ComputeFailed          ComputeState = "failed"
	ComputeCleanupRequired ComputeState = "cleanup_required"
)

const (
	StorageCreating        StorageState = "creating"
	StorageAvailable       StorageState = "available"
	StorageAttaching       StorageState = "attaching"
	StorageAttached        StorageState = "attached"
	StorageDetaching       StorageState = "detaching"
	StorageDetached        StorageState = "detached"
	StorageDestroying      StorageState = "destroying"
	StorageDestroyed       StorageState = "destroyed"
	StorageFailed          StorageState = "failed"
	StorageCleanupRequired StorageState = "cleanup_required"
)

const (
	OperationAccepted       OperationState = "accepted"
	OperationDryRun         OperationState = "dry_run"
	OperationApplying       OperationState = "applying"
	OperationVerifying      OperationState = "verifying"
	OperationSucceeded      OperationState = "succeeded"
	OperationFailed         OperationState = "failed"
	OperationBlocked        OperationState = "blocked"
	OperationNeedsHumanGate OperationState = "needs_human_gate"
)

type ComputeResource struct {
	ID             string
	OwnerAccountID string
	PackageID      string
	State          ComputeState
	ProviderRef    string
}

type StorageVolume struct {
	ID             string
	OwnerAccountID string
	PackageID      string
	State          StorageState
	ProviderRef    string
	SizeGB         int
}

type DestroyStorageRequest struct {
	Confirm     bool
	HumanGateID string
	RequestedBy string
}

type DestroyResult struct {
	Compute ComputeResource
	Storage StorageVolume
}
