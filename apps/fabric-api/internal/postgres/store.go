package postgres

import (
	"context"
	"errors"

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
  ADD COLUMN IF NOT EXISTS owner_account_id TEXT NOT NULL DEFAULT '';
`

type OperationRow struct {
	ID             string
	CorrelationID  string
	IdempotencyKey string
	RequestedBy    string
	ResourceID     string
	ResourceKind   string
	State          string
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

func (s *Store) CreateWorkspaceEntry(ctx context.Context, row WorkspaceEntryRow) error {
	if s == nil || s.pool == nil {
		return ErrStoreNotOpen
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO workspace_entries (id, owner_account_id, workspace_id, attachment_id, state, host, path)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO NOTHING
`, row.ID, row.OwnerAccountID, row.WorkspaceID, row.AttachmentID, row.State, row.Host, row.Path)
	return err
}

func defaultJSON(value string) string {
	if value == "" {
		return "{}"
	}
	return value
}
