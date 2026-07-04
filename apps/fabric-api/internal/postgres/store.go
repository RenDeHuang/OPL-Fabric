package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStoreNotOpen = errors.New("postgres_store_not_open")

type Store struct {
	pool *pgxpool.Pool
}

const migrationSQL = `
ALTER TABLE IF EXISTS storage_attachments
  ADD COLUMN IF NOT EXISTS owner_account_id TEXT NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS workspace_entries
  ADD COLUMN IF NOT EXISTS owner_account_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS service_ref TEXT NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS fabric_operations
  ADD COLUMN IF NOT EXISTS lease_owner TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS provider_refs JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS evidence_refs JSONB NOT NULL DEFAULT '[]'::jsonb;
`

type OperationRow struct {
	ID             string
	CorrelationID  string
	IdempotencyKey string
	RequestedBy    string
	ResourceID     string
	ResourceKind   string
	State          string
	LeaseOwner     string
	Attempts       int
	LastError      string
}

type StorageVolumeRow struct {
	ID              string
	OwnerAccountID  string
	ProductPresetID string
	State           string
	ProviderRef     string
	SizeGB          int
	Retained        bool
}

type ComputeResourceRow struct {
	ID                   string
	OwnerAccountID       string
	ProductPresetID      string
	ComputeShapeJSON     string
	ProviderInstanceType string
	CapacityPoolID       string
	IsolationMode        string
	NodePoolID           string
	RuntimeRef           string
	State                string
	ProviderRef          string
}

type StorageAttachmentRow struct {
	ID             string
	OwnerAccountID string
	ComputeID      string
	StorageID      string
	State          string
	MountPath      string
	ProviderRef    string
}

type WorkspaceEntryRow struct {
	ID             string
	OwnerAccountID string
	WorkspaceID    string
	AttachmentID   string
	State          string
	Host           string
	Path           string
	ServiceRef     string
}

type WorkspaceRow struct {
	ID              string
	OwnerAccountID  string
	WorkspaceName   string
	ProductPresetID string
	StorageID       string
	ComputeID       string
	AttachmentID    string
	EntryID         string
	OperationID     string
	State           string
}

type WorkspaceReservation struct {
	Operation  OperationRow
	Storage    StorageVolumeRow
	Compute    ComputeResourceRow
	Attachment StorageAttachmentRow
	Entry      WorkspaceEntryRow
	Workspace  WorkspaceRow
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Migrate(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	if _, err := s.pool.Exec(ctx, SchemaSQL); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, migrationSQL)
	return err
}

func (s *Store) CreateOperation(ctx context.Context, row OperationRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO fabric_operations (id, correlation_id, idempotency_key, requested_by, resource_id, resource_kind, state)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (idempotency_key) DO NOTHING
`, row.ID, row.CorrelationID, row.IdempotencyKey, row.RequestedBy, row.ResourceID, row.ResourceKind, row.State)
	return err
}

func (s *Store) GetOperation(ctx context.Context, id string) (OperationRow, error) {
	if s == nil || s.pool == nil {
		return OperationRow{}, ErrStoreNotOpen
	}
	var row OperationRow
	err := s.pool.QueryRow(ctx, `
SELECT id, correlation_id, idempotency_key, requested_by, resource_id, resource_kind, state
FROM fabric_operations
WHERE id = $1
`, id).Scan(&row.ID, &row.CorrelationID, &row.IdempotencyKey, &row.RequestedBy, &row.ResourceID, &row.ResourceKind, &row.State)
	return row, err
}

func (s *Store) UpdateOperationState(ctx context.Context, id, state string) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
UPDATE fabric_operations
SET state = $2, updated_at = now()
WHERE id = $1
`, id, state)
	return err
}

func (s *Store) ListAcceptedOperations(ctx context.Context, limit int) ([]OperationRow, error) {
	if s == nil || s.pool == nil {
		return nil, ErrStoreNotOpen
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, correlation_id, idempotency_key, requested_by, resource_id, resource_kind, state, lease_owner, attempts, last_error
FROM fabric_operations
WHERE state = 'accepted'
  AND (lease_expires_at IS NULL OR lease_expires_at <= now())
ORDER BY created_at ASC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	operations := []OperationRow{}
	for rows.Next() {
		var row OperationRow
		if err := rows.Scan(&row.ID, &row.CorrelationID, &row.IdempotencyKey, &row.RequestedBy, &row.ResourceID, &row.ResourceKind, &row.State, &row.LeaseOwner, &row.Attempts, &row.LastError); err != nil {
			return nil, err
		}
		operations = append(operations, row)
	}
	return operations, rows.Err()
}

func (s *Store) LeaseOperation(ctx context.Context, id, owner string, ttl time.Duration) (bool, error) {
	if s == nil || s.pool == nil {
		return false, ErrStoreNotOpen
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	ttlSeconds := int(ttl.Seconds())
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}
	tag, err := s.pool.Exec(ctx, `
UPDATE fabric_operations
SET lease_owner = $2, lease_expires_at = now() + make_interval(secs => $3), updated_at = now()
WHERE id = $1
  AND state = 'accepted'
  AND (lease_expires_at IS NULL OR lease_expires_at <= now())
`, id, owner, ttlSeconds)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) RecordOperationFailure(ctx context.Context, id string, cause error) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	_, err := s.pool.Exec(ctx, `
UPDATE fabric_operations
SET attempts = attempts + 1,
    last_error = $2,
    state = CASE WHEN attempts + 1 >= 3 THEN 'failed' ELSE 'accepted' END,
    lease_owner = '',
    lease_expires_at = NULL,
    updated_at = now()
WHERE id = $1
`, id, message)
	return err
}

func (s *Store) CreateStorageVolume(ctx context.Context, row StorageVolumeRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO storage_volumes (id, owner_account_id, product_preset_id, state, provider_ref, size_gb, retained)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.ProductPresetID, row.State, row.ProviderRef, row.SizeGB, row.Retained)
	return err
}

func (s *Store) GetStorageVolume(ctx context.Context, id string) (StorageVolumeRow, error) {
	if s == nil || s.pool == nil {
		return StorageVolumeRow{}, ErrStoreNotOpen
	}
	var row StorageVolumeRow
	err := s.pool.QueryRow(ctx, `
SELECT id, owner_account_id, product_preset_id, state, provider_ref, size_gb, retained
FROM storage_volumes
WHERE id = $1
`, id).Scan(&row.ID, &row.OwnerAccountID, &row.ProductPresetID, &row.State, &row.ProviderRef, &row.SizeGB, &row.Retained)
	return row, err
}

func (s *Store) UpdateStorageVolume(ctx context.Context, row StorageVolumeRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
UPDATE storage_volumes
SET state = $2, provider_ref = $3, retained = $4, updated_at = now()
WHERE id = $1
`, row.ID, row.State, row.ProviderRef, row.Retained)
	return err
}

func (s *Store) CreateComputeResource(ctx context.Context, row ComputeResourceRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO compute_resources (id, owner_account_id, product_preset_id, compute_shape_json, provider_instance_type, capacity_pool_id, isolation_mode, node_pool_id, runtime_ref, state, provider_ref)
VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.ProductPresetID, defaultJSON(row.ComputeShapeJSON), row.ProviderInstanceType, row.CapacityPoolID, row.IsolationMode, row.NodePoolID, row.RuntimeRef, row.State, row.ProviderRef)
	return err
}

func (s *Store) GetComputeResource(ctx context.Context, id string) (ComputeResourceRow, error) {
	if s == nil || s.pool == nil {
		return ComputeResourceRow{}, ErrStoreNotOpen
	}
	var row ComputeResourceRow
	err := s.pool.QueryRow(ctx, `
SELECT id, owner_account_id, product_preset_id, compute_shape_json::text, provider_instance_type, capacity_pool_id, isolation_mode, node_pool_id, runtime_ref, state, provider_ref
FROM compute_resources
WHERE id = $1
`, id).Scan(&row.ID, &row.OwnerAccountID, &row.ProductPresetID, &row.ComputeShapeJSON, &row.ProviderInstanceType, &row.CapacityPoolID, &row.IsolationMode, &row.NodePoolID, &row.RuntimeRef, &row.State, &row.ProviderRef)
	return row, err
}

func (s *Store) UpdateComputeResource(ctx context.Context, row ComputeResourceRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
UPDATE compute_resources
SET node_pool_id = $2, runtime_ref = $3, state = $4, provider_ref = $5, updated_at = now()
WHERE id = $1
`, row.ID, row.NodePoolID, row.RuntimeRef, row.State, row.ProviderRef)
	return err
}

func (s *Store) CreateStorageAttachment(ctx context.Context, row StorageAttachmentRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO storage_attachments (id, owner_account_id, compute_id, storage_id, state, mount_path, provider_ref)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.ComputeID, row.StorageID, row.State, row.MountPath, row.ProviderRef)
	return err
}

func (s *Store) GetStorageAttachment(ctx context.Context, id string) (StorageAttachmentRow, error) {
	if s == nil || s.pool == nil {
		return StorageAttachmentRow{}, ErrStoreNotOpen
	}
	var row StorageAttachmentRow
	err := s.pool.QueryRow(ctx, `
SELECT id, owner_account_id, compute_id, storage_id, state, mount_path, provider_ref
FROM storage_attachments
WHERE id = $1
`, id).Scan(&row.ID, &row.OwnerAccountID, &row.ComputeID, &row.StorageID, &row.State, &row.MountPath, &row.ProviderRef)
	return row, err
}

func (s *Store) UpdateStorageAttachment(ctx context.Context, row StorageAttachmentRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
UPDATE storage_attachments
SET state = $2, mount_path = $3, provider_ref = $4, updated_at = now()
WHERE id = $1
`, row.ID, row.State, row.MountPath, row.ProviderRef)
	return err
}

func (s *Store) CreateWorkspaceEntry(ctx context.Context, row WorkspaceEntryRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO workspace_entries (id, owner_account_id, workspace_id, attachment_id, state, host, path, service_ref)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.WorkspaceID, row.AttachmentID, row.State, row.Host, row.Path, row.ServiceRef)
	return err
}

func (s *Store) GetWorkspaceEntry(ctx context.Context, id string) (WorkspaceEntryRow, error) {
	if s == nil || s.pool == nil {
		return WorkspaceEntryRow{}, ErrStoreNotOpen
	}
	var row WorkspaceEntryRow
	err := s.pool.QueryRow(ctx, `
SELECT id, owner_account_id, workspace_id, attachment_id, state, host, path, service_ref
FROM workspace_entries
WHERE id = $1
`, id).Scan(&row.ID, &row.OwnerAccountID, &row.WorkspaceID, &row.AttachmentID, &row.State, &row.Host, &row.Path, &row.ServiceRef)
	return row, err
}

func (s *Store) UpdateWorkspaceEntry(ctx context.Context, row WorkspaceEntryRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
UPDATE workspace_entries
SET state = $2, host = $3, path = $4, service_ref = $5, updated_at = now()
WHERE id = $1
`, row.ID, row.State, row.Host, row.Path, row.ServiceRef)
	return err
}

func (s *Store) CreateWorkspace(ctx context.Context, row WorkspaceRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO workspaces (id, owner_account_id, workspace_name, product_preset_id, storage_id, compute_id, attachment_id, entry_id, operation_id, state)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.WorkspaceName, row.ProductPresetID, row.StorageID, row.ComputeID, row.AttachmentID, row.EntryID, row.OperationID, row.State)
	return err
}

func (s *Store) CreateWorkspaceReservation(ctx context.Context, reservation WorkspaceReservation) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if err := insertOperation(ctx, tx, reservation.Operation); err != nil {
		return err
	}
	if err := insertStorageVolume(ctx, tx, reservation.Storage); err != nil {
		return err
	}
	if err := insertComputeResource(ctx, tx, reservation.Compute); err != nil {
		return err
	}
	if err := insertStorageAttachment(ctx, tx, reservation.Attachment); err != nil {
		return err
	}
	if err := insertWorkspaceEntry(ctx, tx, reservation.Entry); err != nil {
		return err
	}
	if err := insertWorkspace(ctx, tx, reservation.Workspace); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) GetWorkspace(ctx context.Context, id string) (WorkspaceRow, error) {
	if s == nil || s.pool == nil {
		return WorkspaceRow{}, ErrStoreNotOpen
	}
	var row WorkspaceRow
	err := s.pool.QueryRow(ctx, `
SELECT id, owner_account_id, workspace_name, product_preset_id, storage_id, compute_id, attachment_id, entry_id, operation_id, state
FROM workspaces
WHERE id = $1
`, id).Scan(&row.ID, &row.OwnerAccountID, &row.WorkspaceName, &row.ProductPresetID, &row.StorageID, &row.ComputeID, &row.AttachmentID, &row.EntryID, &row.OperationID, &row.State)
	return row, err
}

func (s *Store) UpdateWorkspace(ctx context.Context, row WorkspaceRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
UPDATE workspaces
SET state = $2, updated_at = now()
WHERE id = $1
`, row.ID, row.State)
	return err
}

func insertOperation(ctx context.Context, tx pgx.Tx, row OperationRow) error {
	_, err := tx.Exec(ctx, `
INSERT INTO fabric_operations (id, correlation_id, idempotency_key, requested_by, resource_id, resource_kind, state)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (idempotency_key) DO NOTHING
`, row.ID, row.CorrelationID, row.IdempotencyKey, row.RequestedBy, row.ResourceID, row.ResourceKind, row.State)
	return err
}

func insertStorageVolume(ctx context.Context, tx pgx.Tx, row StorageVolumeRow) error {
	_, err := tx.Exec(ctx, `
INSERT INTO storage_volumes (id, owner_account_id, product_preset_id, state, provider_ref, size_gb, retained)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.ProductPresetID, row.State, row.ProviderRef, row.SizeGB, row.Retained)
	return err
}

func insertComputeResource(ctx context.Context, tx pgx.Tx, row ComputeResourceRow) error {
	_, err := tx.Exec(ctx, `
INSERT INTO compute_resources (id, owner_account_id, product_preset_id, compute_shape_json, provider_instance_type, capacity_pool_id, isolation_mode, node_pool_id, runtime_ref, state, provider_ref)
VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.ProductPresetID, defaultJSON(row.ComputeShapeJSON), row.ProviderInstanceType, row.CapacityPoolID, row.IsolationMode, row.NodePoolID, row.RuntimeRef, row.State, row.ProviderRef)
	return err
}

func insertStorageAttachment(ctx context.Context, tx pgx.Tx, row StorageAttachmentRow) error {
	_, err := tx.Exec(ctx, `
INSERT INTO storage_attachments (id, owner_account_id, compute_id, storage_id, state, mount_path, provider_ref)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.ComputeID, row.StorageID, row.State, row.MountPath, row.ProviderRef)
	return err
}

func insertWorkspaceEntry(ctx context.Context, tx pgx.Tx, row WorkspaceEntryRow) error {
	_, err := tx.Exec(ctx, `
INSERT INTO workspace_entries (id, owner_account_id, workspace_id, attachment_id, state, host, path, service_ref)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.WorkspaceID, row.AttachmentID, row.State, row.Host, row.Path, row.ServiceRef)
	return err
}

func insertWorkspace(ctx context.Context, tx pgx.Tx, row WorkspaceRow) error {
	_, err := tx.Exec(ctx, `
INSERT INTO workspaces (id, owner_account_id, workspace_name, product_preset_id, storage_id, compute_id, attachment_id, entry_id, operation_id, state)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.WorkspaceName, row.ProductPresetID, row.StorageID, row.ComputeID, row.AttachmentID, row.EntryID, row.OperationID, row.State)
	return err
}

func defaultJSON(value string) string {
	if value == "" {
		return "{}"
	}
	return value
}
