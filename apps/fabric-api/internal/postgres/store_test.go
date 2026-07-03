package postgres

import (
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
