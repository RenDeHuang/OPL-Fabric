package postgres

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSchemaContainsRequiredTables(t *testing.T) {
	required := []string{
		"compute_allocations",
		"storage_volumes",
		"storage_attachments",
		"workspace_entries",
		"workspaces",
		"fabric_operations",
		"fabric_events",
		"fabric_evidence_refs",
		"human_gates",
		"idempotency_keys",
	}
	for _, table := range required {
		if !strings.Contains(SchemaSQL, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("schema missing table %s", table)
		}
	}
}

func TestSchemaContainsPersistenceConstraints(t *testing.T) {
	required := []string{
		"compute_allocation_id TEXT NOT NULL REFERENCES compute_allocations(id)",
		"storage_id TEXT NOT NULL REFERENCES storage_volumes(id)",
		"attachment_id TEXT NOT NULL REFERENCES storage_attachments(id)",
		"owner_account_id TEXT NOT NULL",
		"service_ref TEXT NOT NULL DEFAULT ''",
		"compute_shape_json JSONB NOT NULL DEFAULT '{}'::jsonb",
		"retained BOOLEAN NOT NULL DEFAULT true",
		"operation_id TEXT NOT NULL REFERENCES fabric_operations(id)",
		"operation_id TEXT NOT NULL UNIQUE REFERENCES fabric_operations(id)",
		"idempotency_key TEXT NOT NULL UNIQUE",
		"lease_owner TEXT NOT NULL DEFAULT ''",
		"lease_expires_at TIMESTAMPTZ",
		"attempts INTEGER NOT NULL DEFAULT 0",
		"last_error TEXT NOT NULL DEFAULT ''",
		"provider_refs JSONB NOT NULL DEFAULT '{}'::jsonb",
		"evidence_refs JSONB NOT NULL DEFAULT '[]'::jsonb",
		"storage_id TEXT NOT NULL REFERENCES storage_volumes(id)",
		"compute_allocation_id TEXT NOT NULL REFERENCES compute_allocations(id)",
		"entry_id TEXT NOT NULL REFERENCES workspace_entries(id)",
		"CHECK (size_gb > 0)",
		"CHECK (sha256 ~ '^[A-Fa-f0-9]{64}$')",
	}
	for _, fragment := range required {
		if !strings.Contains(SchemaSQL, fragment) {
			t.Fatalf("schema missing constraint fragment %q", fragment)
		}
	}
}

func TestMigrationSQLBackfillsPhaseTwoColumns(t *testing.T) {
	required := []string{
		"ALTER TABLE IF EXISTS storage_attachments",
		"ALTER TABLE IF EXISTS workspace_entries",
		"ADD COLUMN IF NOT EXISTS owner_account_id TEXT NOT NULL DEFAULT ''",
		"ADD COLUMN IF NOT EXISTS service_ref TEXT NOT NULL DEFAULT ''",
	}
	for _, fragment := range required {
		if !strings.Contains(migrationSQL, fragment) {
			t.Fatalf("migration SQL missing fragment %q", fragment)
		}
	}
}

func TestNilStoreMigrateReturnsError(t *testing.T) {
	var store *Store
	if err := store.Migrate(context.Background()); err != ErrStoreNotOpen {
		t.Fatalf("error = %v, want %v", err, ErrStoreNotOpen)
	}
}

func TestNilStoreResourceMethodsReturnError(t *testing.T) {
	var store *Store
	ctx := context.Background()
	if err := store.CreateOperation(ctx, OperationRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateOperation error = %v, want %v", err, ErrStoreNotOpen)
	}
	if _, err := store.GetOperation(ctx, "op-1"); err != ErrStoreNotOpen {
		t.Fatalf("GetOperation error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.CreateStorageVolume(ctx, StorageVolumeRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateStorageVolume error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.CreateComputeAllocation(ctx, ComputeAllocationRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateComputeAllocation error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.CreateStorageAttachment(ctx, StorageAttachmentRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateStorageAttachment error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.CreateWorkspaceEntry(ctx, WorkspaceEntryRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateWorkspaceEntry error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.UpdateOperationState(ctx, "op-1", "applying"); err != ErrStoreNotOpen {
		t.Fatalf("UpdateOperationState error = %v, want %v", err, ErrStoreNotOpen)
	}
	if _, err := store.GetStorageVolume(ctx, "storage-1"); err != ErrStoreNotOpen {
		t.Fatalf("GetStorageVolume error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.UpdateStorageVolume(ctx, StorageVolumeRow{}); err != ErrStoreNotOpen {
		t.Fatalf("UpdateStorageVolume error = %v, want %v", err, ErrStoreNotOpen)
	}
	if _, err := store.GetComputeAllocation(ctx, "compute-1"); err != ErrStoreNotOpen {
		t.Fatalf("GetComputeAllocation error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.UpdateComputeAllocation(ctx, ComputeAllocationRow{}); err != ErrStoreNotOpen {
		t.Fatalf("UpdateComputeAllocation error = %v, want %v", err, ErrStoreNotOpen)
	}
	if _, err := store.GetStorageAttachment(ctx, "attach-1"); err != ErrStoreNotOpen {
		t.Fatalf("GetStorageAttachment error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.UpdateStorageAttachment(ctx, StorageAttachmentRow{}); err != ErrStoreNotOpen {
		t.Fatalf("UpdateStorageAttachment error = %v, want %v", err, ErrStoreNotOpen)
	}
	if _, err := store.GetWorkspaceEntry(ctx, "entry-1"); err != ErrStoreNotOpen {
		t.Fatalf("GetWorkspaceEntry error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.UpdateWorkspaceEntry(ctx, WorkspaceEntryRow{}); err != ErrStoreNotOpen {
		t.Fatalf("UpdateWorkspaceEntry error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.CreateWorkspace(ctx, WorkspaceRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateWorkspace error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.CreateWorkspaceReservation(ctx, WorkspaceReservation{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateWorkspaceReservation error = %v, want %v", err, ErrStoreNotOpen)
	}
	if _, err := store.GetWorkspace(ctx, "workspace-1"); err != ErrStoreNotOpen {
		t.Fatalf("GetWorkspace error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.UpdateWorkspace(ctx, WorkspaceRow{}); err != ErrStoreNotOpen {
		t.Fatalf("UpdateWorkspace error = %v, want %v", err, ErrStoreNotOpen)
	}
	if _, err := store.ListAcceptedOperations(ctx, 10); err != ErrStoreNotOpen {
		t.Fatalf("ListAcceptedOperations error = %v, want %v", err, ErrStoreNotOpen)
	}
	if _, err := store.LeaseOperation(ctx, "op-1", "worker-1", time.Minute); err != ErrStoreNotOpen {
		t.Fatalf("LeaseOperation error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.RecordOperationFailure(ctx, "op-1", context.Canceled); err != ErrStoreNotOpen {
		t.Fatalf("RecordOperationFailure error = %v, want %v", err, ErrStoreNotOpen)
	}
}
