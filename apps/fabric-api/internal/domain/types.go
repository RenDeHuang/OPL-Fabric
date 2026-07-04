package domain

type ComputeState string
type StorageState string
type OperationState string

const (
	ComputeCreating        ComputeState = "creating"
	ComputeRunning         ComputeState = "running"
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
	ID                   string
	OwnerAccountID       string
	ProductPresetID      string
	ComputeShapeJSON     string
	ProviderInstanceType string
	CapacityPoolID       string
	IsolationMode        string
	NodePoolID           string
	RuntimeRef           string
	State                ComputeState
	ProviderRef          string
}

type StorageVolume struct {
	ID              string
	OwnerAccountID  string
	ProductPresetID string
	State           StorageState
	ProviderRef     string
	SizeGB          int
	Retained        bool
}

type StorageAttachment struct {
	ID          string
	ComputeID   string
	StorageID   string
	State       StorageState
	MountPath   string
	ProviderRef string
}

type WorkspaceEntry struct {
	ID           string
	WorkspaceID  string
	AttachmentID string
	State        string
	Host         string
	Path         string
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

type FabricOperation struct {
	ID             string         `json:"id"`
	CorrelationID  string         `json:"correlationId"`
	IdempotencyKey string         `json:"idempotencyKey"`
	RequestedBy    string         `json:"requestedBy"`
	ResourceID     string         `json:"resourceId"`
	ResourceKind   string         `json:"resourceKind"`
	State          OperationState `json:"state"`
}

type EvidenceRef struct {
	ID          string `json:"id"`
	OperationID string `json:"operationId"`
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	SHA256      string `json:"sha256"`
}
