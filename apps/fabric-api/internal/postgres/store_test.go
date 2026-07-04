package postgres

import (
	"context"
	"strings"
	"testing"
)

func TestSchemaContainsRequiredTables(t *testing.T) {
	required := []string{
		"compute_resources",
		"storage_volumes",
		"storage_attachments",
		"workspace_entries",
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
		"compute_id TEXT NOT NULL REFERENCES compute_resources(id)",
		"storage_id TEXT NOT NULL REFERENCES storage_volumes(id)",
		"attachment_id TEXT NOT NULL REFERENCES storage_attachments(id)",
		"owner_account_id TEXT NOT NULL",
		"compute_shape_json JSONB NOT NULL DEFAULT '{}'::jsonb",
		"retained BOOLEAN NOT NULL DEFAULT true",
		"operation_id TEXT NOT NULL REFERENCES fabric_operations(id)",
		"operation_id TEXT NOT NULL UNIQUE REFERENCES fabric_operations(id)",
		"idempotency_key TEXT NOT NULL UNIQUE",
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
	if err := store.CreateComputeResource(ctx, ComputeResourceRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateComputeResource error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.CreateStorageAttachment(ctx, StorageAttachmentRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateStorageAttachment error = %v, want %v", err, ErrStoreNotOpen)
	}
	if err := store.CreateWorkspaceEntry(ctx, WorkspaceEntryRow{}); err != ErrStoreNotOpen {
		t.Fatalf("CreateWorkspaceEntry error = %v, want %v", err, ErrStoreNotOpen)
	}
}
