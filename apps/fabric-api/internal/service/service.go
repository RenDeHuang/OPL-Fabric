package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
)

var (
	ErrStoreRequired      = errors.New("fabric_store_required")
	ErrRequestedByMissing = errors.New("requested_by_required")
	ErrAccountIDMissing   = errors.New("account_id_required")
	ErrConfirmationNeeded = errors.New("confirmation_required")
)

type Store interface {
	CreateOperation(context.Context, postgres.OperationRow) error
	GetOperation(context.Context, string) (postgres.OperationRow, error)
	CreateStorageVolume(context.Context, postgres.StorageVolumeRow) error
	CreateComputeResource(context.Context, postgres.ComputeResourceRow) error
	CreateStorageAttachment(context.Context, postgres.StorageAttachmentRow) error
	CreateWorkspaceEntry(context.Context, postgres.WorkspaceEntryRow) error
	CreateWorkspace(context.Context, postgres.WorkspaceRow) error
	GetWorkspace(context.Context, string) (postgres.WorkspaceRow, error)
	GetStorageVolume(context.Context, string) (postgres.StorageVolumeRow, error)
	GetComputeResource(context.Context, string) (postgres.ComputeResourceRow, error)
	GetStorageAttachment(context.Context, string) (postgres.StorageAttachmentRow, error)
	GetWorkspaceEntry(context.Context, string) (postgres.WorkspaceEntryRow, error)
}

type Config struct {
	Catalog             catalog.Catalog
	DatabaseURL         string
	OperatorToken       string
	KubernetesNamespace string
	IngressClass        string
	ImagePullSecretName string
	WorkspaceImage      string
	WorkspaceDomain     string
	StorageClass        string
	TencentTKERegion    string
	TencentClusterID    string
	TencentSecretID     string
	TencentSecretKey    string
	TencentTCRRegistry  string
	TencentTCRNamespace string
	TencentTCRRegion    string
	Store               Store
}

type Service struct {
	catalog             catalog.Catalog
	databaseURL         string
	operatorToken       string
	kubernetesNamespace string
	ingressClass        string
	imagePullSecretName string
	workspaceImage      string
	workspaceDomain     string
	storageClass        string
	tencentTKERegion    string
	tencentClusterID    string
	tencentSecretID     string
	tencentSecretKey    string
	tencentTCRRegistry  string
	tencentTCRNamespace string
	tencentTCRRegion    string
	store               Store
}

type Readiness struct {
	Ready           bool            `json:"ready"`
	Provider        string          `json:"provider"`
	MissingEnv      []string        `json:"missingEnv"`
	ResourceCatalog catalog.Catalog `json:"resourceCatalog"`
	Blockers        []string        `json:"blockers"`
	RepairHints     []string        `json:"repairHints"`
}

func New(cfg Config) *Service {
	return &Service{
		catalog:             cfg.Catalog,
		databaseURL:         cfg.DatabaseURL,
		operatorToken:       cfg.OperatorToken,
		kubernetesNamespace: cfg.KubernetesNamespace,
		ingressClass:        cfg.IngressClass,
		imagePullSecretName: cfg.ImagePullSecretName,
		workspaceImage:      cfg.WorkspaceImage,
		workspaceDomain:     cfg.WorkspaceDomain,
		storageClass:        cfg.StorageClass,
		tencentTKERegion:    cfg.TencentTKERegion,
		tencentClusterID:    cfg.TencentClusterID,
		tencentSecretID:     cfg.TencentSecretID,
		tencentSecretKey:    cfg.TencentSecretKey,
		tencentTCRRegistry:  cfg.TencentTCRRegistry,
		tencentTCRNamespace: cfg.TencentTCRNamespace,
		tencentTCRRegion:    cfg.TencentTCRRegion,
		store:               cfg.Store,
	}
}

func (s *Service) Catalog() catalog.Catalog {
	return s.catalog
}

func (s *Service) Readiness() Readiness {
	missingEnv := s.missingEnv()
	return Readiness{
		Ready:           len(missingEnv) == 0,
		Provider:        "tencent-tke",
		MissingEnv:      missingEnv,
		ResourceCatalog: s.catalog,
		Blockers:        blockersForMissingEnv(missingEnv),
		RepairHints:     repairHintsForMissingEnv(missingEnv),
	}
}

type MutationHeaders struct {
	IdempotencyKey string
	CorrelationID  string
}

type OperationReceipt struct {
	OperationID  string `json:"operationId"`
	State        string `json:"state"`
	ResourceKind string `json:"resourceKind"`
	ResourceID   string `json:"resourceId"`
}

type CreateStorageVolumeRequest struct {
	AccountID       string `json:"accountId"`
	RequestedBy     string `json:"requestedBy"`
	ProductPresetID string `json:"productPresetId"`
	SizeGB          int    `json:"sizeGb"`
}

type CreateComputeResourceRequest struct {
	AccountID            string         `json:"accountId"`
	RequestedBy          string         `json:"requestedBy"`
	ProductPresetID      string         `json:"productPresetId"`
	ComputeShape         map[string]any `json:"computeShape"`
	ProviderInstanceType string         `json:"providerInstanceType"`
	CapacityPoolID       string         `json:"capacityPoolId"`
	IsolationMode        string         `json:"isolationMode"`
}

type CreateStorageAttachmentRequest struct {
	AccountID   string `json:"accountId"`
	RequestedBy string `json:"requestedBy"`
	ComputeID   string `json:"computeId"`
	StorageID   string `json:"storageId"`
	MountPath   string `json:"mountPath"`
}

type CreateWorkspaceEntryRequest struct {
	AccountID     string `json:"accountId"`
	RequestedBy   string `json:"requestedBy"`
	WorkspaceName string `json:"workspaceName"`
	AttachmentID  string `json:"attachmentId"`
}

type CreateWorkspaceRequest struct {
	AccountID            string         `json:"accountId"`
	RequestedBy          string         `json:"requestedBy"`
	WorkspaceName        string         `json:"workspaceName"`
	ProductPresetID      string         `json:"productPresetId"`
	ComputeShape         map[string]any `json:"computeShape"`
	ProviderInstanceType string         `json:"providerInstanceType"`
	CapacityPoolID       string         `json:"capacityPoolId"`
	IsolationMode        string         `json:"isolationMode"`
	Storage              struct {
		SizeGB int `json:"sizeGb"`
	} `json:"storage"`
}

type Workspace struct {
	WorkspaceID string                 `json:"workspaceId"`
	State       string                 `json:"state"`
	Storage     WorkspaceStorageStatus `json:"storage"`
	Compute     WorkspaceComputeStatus `json:"compute"`
	Attachment  WorkspaceAttachStatus  `json:"attachment"`
	Entry       WorkspaceEntryStatus   `json:"entry"`
	OperationID string                 `json:"operationId"`
}

type WorkspaceStorageStatus struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	SizeGB int    `json:"sizeGb"`
}

type WorkspaceComputeStatus struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

type WorkspaceAttachStatus struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	MountPath string `json:"mountPath"`
}

type WorkspaceEntryStatus struct {
	ID    string `json:"id"`
	State string `json:"state"`
	URL   string `json:"url"`
}

type ConfirmRequest struct {
	RequestedBy string `json:"requestedBy"`
	Confirm     bool   `json:"confirm"`
}

func (s *Service) AcceptStorageVolume(ctx context.Context, headers MutationHeaders, req CreateStorageVolumeRequest) (OperationReceipt, error) {
	if err := s.requireMutation(req.AccountID, req.RequestedBy); err != nil {
		return OperationReceipt{}, err
	}
	resourceID := stableID("storage", headers.IdempotencyKey)
	sizeGB := req.SizeGB
	if sizeGB <= 0 {
		sizeGB = 10
	}
	if err := s.store.CreateStorageVolume(ctx, postgres.StorageVolumeRow{
		ID:              resourceID,
		OwnerAccountID:  req.AccountID,
		ProductPresetID: req.ProductPresetID,
		State:           "creating",
		SizeGB:          sizeGB,
		Retained:        true,
	}); err != nil {
		return OperationReceipt{}, err
	}
	return s.acceptOperation(ctx, headers, req.RequestedBy, "storage_volume", resourceID)
}

func (s *Service) AcceptComputeResource(ctx context.Context, headers MutationHeaders, req CreateComputeResourceRequest) (OperationReceipt, error) {
	if err := s.requireMutation(req.AccountID, req.RequestedBy); err != nil {
		return OperationReceipt{}, err
	}
	resourceID := stableID("compute", headers.IdempotencyKey)
	shapeJSON, err := json.Marshal(req.ComputeShape)
	if err != nil {
		return OperationReceipt{}, err
	}
	if err := s.store.CreateComputeResource(ctx, postgres.ComputeResourceRow{
		ID:                   resourceID,
		OwnerAccountID:       req.AccountID,
		ProductPresetID:      req.ProductPresetID,
		ComputeShapeJSON:     string(shapeJSON),
		ProviderInstanceType: req.ProviderInstanceType,
		CapacityPoolID:       req.CapacityPoolID,
		IsolationMode:        req.IsolationMode,
		State:                "creating",
	}); err != nil {
		return OperationReceipt{}, err
	}
	return s.acceptOperation(ctx, headers, req.RequestedBy, "compute_resource", resourceID)
}

func (s *Service) AcceptStorageAttachment(ctx context.Context, headers MutationHeaders, req CreateStorageAttachmentRequest) (OperationReceipt, error) {
	if err := s.requireMutation(req.AccountID, req.RequestedBy); err != nil {
		return OperationReceipt{}, err
	}
	resourceID := stableID("attach", headers.IdempotencyKey)
	mountPath := req.MountPath
	if mountPath == "" {
		mountPath = "/data"
	}
	if err := s.store.CreateStorageAttachment(ctx, postgres.StorageAttachmentRow{
		ID:             resourceID,
		OwnerAccountID: req.AccountID,
		ComputeID:      req.ComputeID,
		StorageID:      req.StorageID,
		State:          "attaching",
		MountPath:      mountPath,
	}); err != nil {
		return OperationReceipt{}, err
	}
	return s.acceptOperation(ctx, headers, req.RequestedBy, "storage_attachment", resourceID)
}

func (s *Service) AcceptWorkspaceEntry(ctx context.Context, headers MutationHeaders, req CreateWorkspaceEntryRequest) (OperationReceipt, error) {
	if err := s.requireMutation(req.AccountID, req.RequestedBy); err != nil {
		return OperationReceipt{}, err
	}
	resourceID := stableID("workspace", headers.IdempotencyKey)
	workspaceID := stableID("ws", headers.IdempotencyKey)
	if err := s.store.CreateWorkspaceEntry(ctx, postgres.WorkspaceEntryRow{
		ID:             resourceID,
		OwnerAccountID: req.AccountID,
		WorkspaceID:    workspaceID,
		AttachmentID:   req.AttachmentID,
		State:          "creating",
		Host:           s.workspaceDomain,
		Path:           "/w/" + workspaceID + "/",
	}); err != nil {
		return OperationReceipt{}, err
	}
	return s.acceptOperation(ctx, headers, req.RequestedBy, "workspace_entry", resourceID)
}

func (s *Service) AcceptWorkspace(ctx context.Context, headers MutationHeaders, req CreateWorkspaceRequest) (OperationReceipt, error) {
	if err := s.requireMutation(req.AccountID, req.RequestedBy); err != nil {
		return OperationReceipt{}, err
	}
	workspaceID := stableID("ws", headers.IdempotencyKey)
	storageID := stableID("storage", headers.IdempotencyKey)
	computeID := stableID("compute", headers.IdempotencyKey)
	attachmentID := stableID("attach", headers.IdempotencyKey)
	entryID := stableID("entry", headers.IdempotencyKey)
	operationID := stableID("op", headers.IdempotencyKey)
	sizeGB := req.Storage.SizeGB
	if sizeGB <= 0 {
		sizeGB = 10
	}
	productPresetID := defaultString(req.ProductPresetID, "basic")
	shapeJSON, err := json.Marshal(req.ComputeShape)
	if err != nil {
		return OperationReceipt{}, err
	}
	if err := s.store.CreateOperation(ctx, postgres.OperationRow{
		ID:             operationID,
		CorrelationID:  headers.CorrelationID,
		IdempotencyKey: headers.IdempotencyKey,
		RequestedBy:    req.RequestedBy,
		ResourceID:     workspaceID,
		ResourceKind:   "workspace",
		State:          "accepted",
	}); err != nil {
		return OperationReceipt{}, err
	}
	if err := s.store.CreateStorageVolume(ctx, postgres.StorageVolumeRow{ID: storageID, OwnerAccountID: req.AccountID, ProductPresetID: productPresetID, State: "creating", SizeGB: sizeGB, Retained: true}); err != nil {
		return OperationReceipt{}, err
	}
	if err := s.store.CreateComputeResource(ctx, postgres.ComputeResourceRow{ID: computeID, OwnerAccountID: req.AccountID, ProductPresetID: productPresetID, ComputeShapeJSON: string(shapeJSON), ProviderInstanceType: req.ProviderInstanceType, CapacityPoolID: req.CapacityPoolID, IsolationMode: req.IsolationMode, State: "creating"}); err != nil {
		return OperationReceipt{}, err
	}
	if err := s.store.CreateStorageAttachment(ctx, postgres.StorageAttachmentRow{ID: attachmentID, OwnerAccountID: req.AccountID, ComputeID: computeID, StorageID: storageID, State: "attaching", MountPath: "/data"}); err != nil {
		return OperationReceipt{}, err
	}
	if err := s.store.CreateWorkspaceEntry(ctx, postgres.WorkspaceEntryRow{ID: entryID, OwnerAccountID: req.AccountID, WorkspaceID: workspaceID, AttachmentID: attachmentID, State: "creating", Host: s.workspaceDomain, Path: "/w/" + workspaceID + "/"}); err != nil {
		return OperationReceipt{}, err
	}
	if err := s.store.CreateWorkspace(ctx, postgres.WorkspaceRow{ID: workspaceID, OwnerAccountID: req.AccountID, WorkspaceName: req.WorkspaceName, ProductPresetID: productPresetID, StorageID: storageID, ComputeID: computeID, AttachmentID: attachmentID, EntryID: entryID, OperationID: operationID, State: "provisioning"}); err != nil {
		return OperationReceipt{}, err
	}
	return OperationReceipt{OperationID: operationID, State: "accepted", ResourceKind: "workspace", ResourceID: workspaceID}, nil
}

func (s *Service) AcceptComputeDestroy(ctx context.Context, headers MutationHeaders, resourceID string, req ConfirmRequest) (OperationReceipt, error) {
	return s.acceptConfirmed(ctx, headers, req, "compute_destroy", resourceID)
}

func (s *Service) AcceptStorageDestroy(ctx context.Context, headers MutationHeaders, resourceID string, req ConfirmRequest) (OperationReceipt, error) {
	return s.acceptConfirmed(ctx, headers, req, "storage_destroy", resourceID)
}

func (s *Service) AcceptAttachmentDetach(ctx context.Context, headers MutationHeaders, resourceID string, req ConfirmRequest) (OperationReceipt, error) {
	return s.acceptConfirmed(ctx, headers, req, "attachment_detach", resourceID)
}

func (s *Service) Operation(ctx context.Context, id string) (OperationReceipt, error) {
	if s.store == nil {
		return OperationReceipt{}, ErrStoreRequired
	}
	row, err := s.store.GetOperation(ctx, id)
	if err != nil {
		return OperationReceipt{}, err
	}
	return OperationReceipt{OperationID: row.ID, State: row.State, ResourceKind: row.ResourceKind, ResourceID: row.ResourceID}, nil
}

func (s *Service) Workspace(ctx context.Context, id string) (Workspace, error) {
	if s.store == nil {
		return Workspace{}, ErrStoreRequired
	}
	workspace, err := s.store.GetWorkspace(ctx, id)
	if err != nil {
		return Workspace{}, err
	}
	storage, err := s.store.GetStorageVolume(ctx, workspace.StorageID)
	if err != nil {
		return Workspace{}, err
	}
	compute, err := s.store.GetComputeResource(ctx, workspace.ComputeID)
	if err != nil {
		return Workspace{}, err
	}
	attachment, err := s.store.GetStorageAttachment(ctx, workspace.AttachmentID)
	if err != nil {
		return Workspace{}, err
	}
	entry, err := s.store.GetWorkspaceEntry(ctx, workspace.EntryID)
	if err != nil {
		return Workspace{}, err
	}
	return Workspace{
		WorkspaceID: workspace.ID,
		State:       workspace.State,
		Storage:     WorkspaceStorageStatus{ID: storage.ID, State: storage.State, SizeGB: storage.SizeGB},
		Compute:     WorkspaceComputeStatus{ID: compute.ID, State: compute.State},
		Attachment:  WorkspaceAttachStatus{ID: attachment.ID, State: attachment.State, MountPath: attachment.MountPath},
		Entry:       WorkspaceEntryStatus{ID: entry.ID, State: entry.State, URL: "https://" + entry.Host + entry.Path},
		OperationID: workspace.OperationID,
	}, nil
}

func (s *Service) acceptConfirmed(ctx context.Context, headers MutationHeaders, req ConfirmRequest, resourceKind, resourceID string) (OperationReceipt, error) {
	if req.RequestedBy == "" {
		return OperationReceipt{}, ErrRequestedByMissing
	}
	if !req.Confirm {
		return OperationReceipt{}, ErrConfirmationNeeded
	}
	return s.acceptOperation(ctx, headers, req.RequestedBy, resourceKind, resourceID)
}

func (s *Service) requireMutation(accountID, requestedBy string) error {
	if s.store == nil {
		return ErrStoreRequired
	}
	if accountID == "" {
		return ErrAccountIDMissing
	}
	if requestedBy == "" {
		return ErrRequestedByMissing
	}
	return nil
}

func (s *Service) acceptOperation(ctx context.Context, headers MutationHeaders, requestedBy, resourceKind, resourceID string) (OperationReceipt, error) {
	if s.store == nil {
		return OperationReceipt{}, ErrStoreRequired
	}
	operationID := stableID("op", headers.IdempotencyKey)
	if err := s.store.CreateOperation(ctx, postgres.OperationRow{
		ID:             operationID,
		CorrelationID:  headers.CorrelationID,
		IdempotencyKey: headers.IdempotencyKey,
		RequestedBy:    requestedBy,
		ResourceID:     resourceID,
		ResourceKind:   resourceKind,
		State:          "accepted",
	}); err != nil {
		return OperationReceipt{}, err
	}
	return OperationReceipt{OperationID: operationID, State: "accepted", ResourceKind: resourceKind, ResourceID: resourceID}, nil
}

func stableID(prefix, key string) string {
	sum := sha256.Sum256([]byte(prefix + ":" + key))
	return prefix + "-" + hex.EncodeToString(sum[:])[:16]
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (s *Service) missingEnv() []string {
	missing := []string{}
	if s.databaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if s.operatorToken == "" {
		missing = append(missing, "OPL_OPERATOR_TOKEN")
	}
	if s.kubernetesNamespace == "" {
		missing = append(missing, "OPL_K8S_NAMESPACE")
	}
	if s.ingressClass == "" {
		missing = append(missing, "OPL_INGRESS_CLASS")
	}
	if s.imagePullSecretName == "" {
		missing = append(missing, "OPL_IMAGE_PULL_SECRET_NAME")
	}
	if s.workspaceImage == "" {
		missing = append(missing, "OPL_WORKSPACE_IMAGE")
	}
	if s.workspaceDomain == "" {
		missing = append(missing, "OPL_WORKSPACE_DOMAIN")
	}
	if s.storageClass == "" {
		missing = append(missing, "OPL_WORKSPACE_STORAGE_CLASS")
	}
	if s.tencentTKERegion == "" {
		missing = append(missing, "TENCENT_TKE_REGION")
	}
	if s.tencentClusterID == "" {
		missing = append(missing, "TENCENT_DEPLOY_CLUSTER_ID")
	}
	if s.tencentSecretID == "" {
		missing = append(missing, "TENCENT_MUTATION_SECRET_ID")
	}
	if s.tencentSecretKey == "" {
		missing = append(missing, "TENCENT_MUTATION_SECRET_KEY")
	}
	if s.tencentTCRRegistry == "" {
		missing = append(missing, "TENCENT_TCR_REGISTRY")
	}
	if s.tencentTCRNamespace == "" {
		missing = append(missing, "TENCENT_TCR_NAMESPACE")
	}
	if s.tencentTCRRegion == "" {
		missing = append(missing, "TENCENT_TCR_REGION")
	}
	return missing
}

func blockersForMissingEnv(missingEnv []string) []string {
	blockers := make([]string, 0, len(missingEnv))
	for _, name := range missingEnv {
		blockers = append(blockers, "missing_env:"+name)
	}
	return blockers
}

func repairHintsForMissingEnv(missingEnv []string) []string {
	hints := make([]string, 0, len(missingEnv))
	for _, name := range missingEnv {
		hints = append(hints, "set "+name+" before serving Fabric traffic")
	}
	return hints
}
