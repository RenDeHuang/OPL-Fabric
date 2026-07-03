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
		"workspace_routes",
		"storage_backups",
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

func TestNilStoreMigrateReturnsError(t *testing.T) {
	var store *Store
	if err := store.Migrate(context.Background()); err != ErrStoreNotOpen {
		t.Fatalf("error = %v, want %v", err, ErrStoreNotOpen)
	}
}
