# OPL Fabric Contract-First Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first runnable contract-first OPL Fabric service with Go API, PostgreSQL schema, client-go provider boundary, JSON contracts, and React TypeScript operator console.

**Architecture:** Start with contracts and pure domain rules, then add persistence, provider interfaces, HTTP endpoints, and the operator console. Kubernetes execution is abstracted behind a provider interface and tested with fake clients before any real-cluster integration is attempted.

**Tech Stack:** Go 1.22, PostgreSQL, Kubernetes `client-go`, React, TypeScript, Vite, Node 22, npm 10.

---

## File Structure

Create these files and directories:

```text
.gitignore
README.md
go.mod
package.json

contracts/fabric-api.openapi.json
contracts/fabric-event-envelope.schema.json
contracts/fabric-human-gate.schema.json
contracts/fabric-lifecycle-ledger.schema.json
contracts/fabric-resource-catalog.schema.json
contracts/fabric-runtime-supervision.schema.json

docs/architecture.md
docs/decisions.md
docs/invariants.md
docs/status.md

apps/fabric-api/go.mod
apps/fabric-api/cmd/fabric-api/main.go
apps/fabric-api/internal/catalog/catalog.go
apps/fabric-api/internal/catalog/catalog_test.go
apps/fabric-api/internal/config/config.go
apps/fabric-api/internal/domain/types.go
apps/fabric-api/internal/domain/rules.go
apps/fabric-api/internal/domain/rules_test.go
apps/fabric-api/internal/evidence/evidence.go
apps/fabric-api/internal/evidence/evidence_test.go
apps/fabric-api/internal/http/server.go
apps/fabric-api/internal/http/server_test.go
apps/fabric-api/internal/k8s/provider.go
apps/fabric-api/internal/k8s/provider_test.go
apps/fabric-api/internal/postgres/migrations.go
apps/fabric-api/internal/postgres/schema.sql
apps/fabric-api/internal/postgres/store.go
apps/fabric-api/internal/service/service.go

apps/fabric-console/index.html
apps/fabric-console/package.json
apps/fabric-console/tsconfig.json
apps/fabric-console/vite.config.ts
apps/fabric-console/src/api.ts
apps/fabric-console/src/App.tsx
apps/fabric-console/src/main.tsx
apps/fabric-console/src/styles.css

deploy/k8s/opl-fabric-api.yaml
deploy/migrations/README.md
```

## Task 1: Repository Foundation

**Files:**
- Create: `.gitignore`
- Create: `README.md`
- Create: `package.json`
- Create: `go.mod`
- Create: `docs/architecture.md`
- Create: `docs/decisions.md`
- Create: `docs/invariants.md`
- Create: `docs/status.md`

- [ ] **Step 1: Create root metadata files**

Write `.gitignore`:

```gitignore
.env
.env.*
!.env.example
node_modules/
dist/
coverage/
.runtime/
*.log
*.test
bin/
tmp/
```

Write root `package.json`:

```json
{
  "name": "opl-fabric",
  "version": "0.1.0",
  "private": true,
  "description": "Contract-first OPL Fabric resource control plane.",
  "scripts": {
    "test": "npm run test:contracts && npm run test:go && npm run test:console",
    "test:contracts": "test ! -d tests/contracts || node --test \"tests/contracts/**/*.test.mjs\"",
    "test:go": "test ! -f apps/fabric-api/go.mod || (cd apps/fabric-api && go test ./...)",
    "test:console": "test ! -f apps/fabric-console/package.json || (npm --prefix apps/fabric-console run typecheck && npm --prefix apps/fabric-console run build)",
    "build": "test ! -f apps/fabric-console/package.json || npm --prefix apps/fabric-console run build"
  }
}
```

Write root `go.mod`:

```go
module github.com/RenDeHuang/OPL-Fabric

go 1.22.2
```

- [ ] **Step 2: Create public repository docs**

Write `README.md`:

````markdown
# OPL Fabric

OPL Fabric is the Fabric control-plane service for OPL Cloud.

It owns resource catalog, provider readiness, compute lifecycle, storage lifecycle, storage attachment, Workspace routing, backup and restore mechanics, Kubernetes provider execution, and Fabric evidence.

It does not own OPL Console commercial flows, wallet and billing truth, OPL Ledger reconciliation, OPL Gateway AI routing, one-person-lab framework internals, or one-person-lab-app WebUI behavior.

## Stack

- Frontend: React + TypeScript
- Backend: Go
- DB: PostgreSQL
- Kubernetes: Go client-go

## Local Verification

The root verification command skips workspaces that have not been created yet and runs their checks once they exist.

```bash
npm test
```
````

Write `docs/architecture.md`:

```markdown
# Architecture

OPL Fabric is an independent service consumed by OPL Console through published HTTP APIs and JSON contracts.

The backend is split into pure domain rules, service orchestration, PostgreSQL persistence, Kubernetes provider execution, evidence recording, and HTTP transport.

The operator console is a React TypeScript UI for readiness, resource status, operation history, and evidence. It is not the commercial OPL Console.
```

Write `docs/invariants.md`:

```markdown
# Invariants

- Fabric owns only Fabric resource truth and provider execution evidence.
- Compute destruction never destroys storage.
- Storage destruction requires explicit confirmation or a recorded human gate.
- Backup deletion never deletes source storage, restored storage, or compute.
- Restore creates new storage and never overwrites an existing PVC.
- Normal provider execution uses Go client-go, not kubectl subprocesses.
- Machine-readable truth lives in contracts.
```

Write `docs/status.md`:

```markdown
# Status

Current status: contract-first foundation implementation.

Supported in the first implementation:

- Fabric contracts.
- Go API process.
- Catalog and readiness endpoints.
- Domain safety rules.
- PostgreSQL schema.
- Kubernetes provider boundary with fake-client tests.
- Operator console build.

Not supported in the first implementation:

- Public GA operations.
- External payment settlement.
- Full OPL Gateway surface.
- Standalone OPL Ledger.
- GPU Workspace exposure.
```

Write `docs/decisions.md`:

```markdown
# Decisions

## 2026-07-03: OPL Fabric split

OPL Fabric is split as an independent Fabric service, not a full OPL Cloud rewrite.

The implementation is contract-first. The backend is Go, persistence is PostgreSQL, Kubernetes operations use client-go, and the frontend is React TypeScript.

one-person-lab is used as the development framework reference for contracts, lifecycle ledgers, evidence, human gates, readiness, and ownership boundaries.
```

- [ ] **Step 3: Verify repository metadata**

Run:

```bash
git diff --check
```

Expected: command exits `0`.

- [ ] **Step 4: Commit**

```bash
git add .gitignore README.md package.json go.mod docs/architecture.md docs/decisions.md docs/invariants.md docs/status.md
git commit -m "chore: initialize opl fabric repository"
```

## Task 2: Contract Schemas

**Files:**
- Create: `contracts/fabric-resource-catalog.schema.json`
- Create: `contracts/fabric-event-envelope.schema.json`
- Create: `contracts/fabric-lifecycle-ledger.schema.json`
- Create: `contracts/fabric-human-gate.schema.json`
- Create: `contracts/fabric-runtime-supervision.schema.json`
- Create: `contracts/fabric-api.openapi.json`
- Create: `tests/contracts/contracts.test.mjs`

- [ ] **Step 1: Add contract validation test first**

Create `tests/contracts/contracts.test.mjs`:

```js
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const contractFiles = [
  "contracts/fabric-api.openapi.json",
  "contracts/fabric-event-envelope.schema.json",
  "contracts/fabric-human-gate.schema.json",
  "contracts/fabric-lifecycle-ledger.schema.json",
  "contracts/fabric-resource-catalog.schema.json",
  "contracts/fabric-runtime-supervision.schema.json"
];

test("contract files are valid JSON with stable identifiers", () => {
  for (const file of contractFiles) {
    const parsed = JSON.parse(readFileSync(file, "utf8"));
    assert.equal(typeof parsed, "object", file);
    assert.ok(parsed.$id || parsed.openapi, `${file} must expose $id or openapi`);
  }
});

test("OpenAPI path templates declare matching path parameters", () => {
  const openapi = JSON.parse(readFileSync("contracts/fabric-api.openapi.json", "utf8"));
  for (const [path, pathItem] of Object.entries(openapi.paths)) {
    const templateParams = [...path.matchAll(/{([^}]+)}/g)].map((match) => match[1]);
    if (templateParams.length === 0) continue;

    for (const [method, operation] of Object.entries(pathItem)) {
      const parameters = operation.parameters || [];
      for (const paramName of templateParams) {
        assert.ok(
          parameters.some((parameter) => parameter.in === "path" && parameter.name === paramName && parameter.required === true),
          `${method.toUpperCase()} ${path} must declare required path parameter ${paramName}`
        );
      }
    }
  }
});

test("OpenAPI operations expose generator-friendly metadata", () => {
  const openapi = JSON.parse(readFileSync("contracts/fabric-api.openapi.json", "utf8"));
  assert.ok(Array.isArray(openapi.servers) && openapi.servers.length > 0, "OpenAPI must declare at least one server");
  assert.ok(Array.isArray(openapi.security) && openapi.security.length > 0, "OpenAPI must declare root security");

  for (const [path, pathItem] of Object.entries(openapi.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      const label = `${method.toUpperCase()} ${path}`;
      assert.equal(typeof operation.operationId, "string", `${label} must declare operationId`);
      assert.ok(operation.operationId.length > 0, `${label} operationId must not be empty`);
      assert.equal(typeof operation.summary, "string", `${label} must declare summary`);
      assert.ok(operation.summary.length > 0, `${label} summary must not be empty`);
      assert.ok(
        Object.keys(operation.responses || {}).some((status) => /^4\d\d$/.test(status)),
        `${label} must declare a 4xx response`
      );
    }
  }
});
```

- [ ] **Step 2: Run contract test and confirm failure**

Run:

```bash
npm run test:contracts
```

Expected: FAIL because the contract files do not exist yet.

- [ ] **Step 3: Add Fabric resource catalog schema**

Create `contracts/fabric-resource-catalog.schema.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://opl.fabric/contracts/fabric-resource-catalog.schema.json",
  "title": "OPL Fabric Resource Catalog",
  "type": "object",
  "additionalProperties": false,
  "required": ["schemaVersion", "owner", "workspacePackages", "computeProfiles", "storageClasses", "workspaceImages", "ingressDomains"],
  "properties": {
    "schemaVersion": { "const": 1 },
    "owner": { "const": "OPL Fabric" },
    "workspacePackages": { "type": "array", "items": { "$ref": "#/$defs/workspacePackage" } },
    "computeProfiles": { "type": "array", "items": { "$ref": "#/$defs/computeProfile" } },
    "storageClasses": { "type": "array", "items": { "$ref": "#/$defs/storageClass" } },
    "workspaceImages": { "type": "array", "items": { "$ref": "#/$defs/workspaceImage" } },
    "ingressDomains": { "type": "array", "items": { "$ref": "#/$defs/ingressDomain" } }
  },
  "$defs": {
    "workspacePackage": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "name", "accelerator", "cpu", "memoryGb", "gpu", "server", "diskGb", "available", "computeProfileId", "storageClassId", "workspaceImageId", "ingressDomainId"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "name": { "type": "string", "minLength": 1 },
        "accelerator": { "enum": ["cpu", "gpu"] },
        "cpu": { "type": "integer", "minimum": 1 },
        "memoryGb": { "type": "integer", "minimum": 1 },
        "gpu": { "type": "integer", "minimum": 0 },
        "server": { "type": "string", "minLength": 1 },
        "diskGb": { "type": "integer", "minimum": 1 },
        "available": { "type": "boolean" },
        "unavailableReason": { "type": "string", "minLength": 1 },
        "computeProfileId": { "type": "string", "minLength": 1 },
        "storageClassId": { "type": "string", "minLength": 1 },
        "workspaceImageId": { "type": "string", "minLength": 1 },
        "ingressDomainId": { "type": "string", "minLength": 1 }
      }
    },
    "computeProfile": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "accelerator", "cpu", "memoryGb", "gpu", "available", "provider"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "accelerator": { "enum": ["cpu", "gpu"] },
        "cpu": { "type": "integer", "minimum": 1 },
        "memoryGb": { "type": "integer", "minimum": 1 },
        "gpu": { "type": "integer", "minimum": 0 },
        "available": { "type": "boolean" },
        "provider": { "type": "string", "minLength": 1 },
        "unavailableReason": { "type": "string", "minLength": 1 }
      }
    },
    "storageClass": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "provider", "storageClassName", "accessMode", "available"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "provider": { "type": "string", "minLength": 1 },
        "storageClassName": { "type": "string", "minLength": 1 },
        "accessMode": { "type": "string", "minLength": 1 },
        "available": { "type": "boolean" }
      }
    },
    "workspaceImage": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "image", "port", "persistentMounts", "available"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "image": { "type": "string", "minLength": 1 },
        "port": { "type": "integer", "minimum": 1 },
        "persistentMounts": { "type": "array", "items": { "type": "string", "minLength": 1 } },
        "available": { "type": "boolean" }
      }
    },
    "ingressDomain": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "host", "pathPattern", "available"],
      "properties": {
        "id": { "type": "string", "minLength": 1 },
        "host": { "type": "string", "minLength": 1 },
        "pathPattern": { "const": "/w/<workspaceId>" },
        "available": { "type": "boolean" }
      }
    }
  }
}
```

- [ ] **Step 4: Add framework-style Fabric schemas**

Create the remaining schema files with these top-level identities:

`contracts/fabric-event-envelope.schema.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://opl.fabric/contracts/fabric-event-envelope.schema.json",
  "title": "OPL Fabric Event Envelope",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "eventId", "eventName", "eventTime", "producer", "correlation", "payload"],
  "properties": {
    "version": { "const": "fabric-event-envelope.v1" },
    "eventId": { "type": "string", "minLength": 1 },
    "eventName": { "type": "string", "minLength": 1 },
    "eventTime": { "type": "string", "format": "date-time" },
    "producer": { "type": "string", "minLength": 1 },
    "correlation": {
      "type": "object",
      "additionalProperties": false,
      "required": ["correlationId", "operationId"],
      "properties": {
        "correlationId": { "type": "string", "minLength": 1 },
        "operationId": { "type": "string", "minLength": 1 },
        "resourceId": { "type": "string", "minLength": 1 }
      }
    },
    "payload": { "type": "object" },
    "evidenceRefs": { "type": "array", "items": { "type": "string", "minLength": 1 } }
  }
}
```

`contracts/fabric-lifecycle-ledger.schema.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://opl.fabric/contracts/fabric-lifecycle-ledger.schema.json",
  "title": "OPL Fabric Lifecycle Ledger",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "operationId", "phase", "status", "summary", "actions"],
  "properties": {
    "version": { "const": "fabric-lifecycle-ledger.v1" },
    "operationId": { "type": "string", "minLength": 1 },
    "phase": { "enum": ["dry_run", "apply", "verify"] },
    "status": { "type": "string", "minLength": 1 },
    "summary": { "type": "string", "minLength": 1 },
    "actions": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["actionId", "actionKind", "targetRef", "result", "sha256"],
        "properties": {
          "actionId": { "type": "string", "minLength": 1 },
          "actionKind": { "type": "string", "minLength": 1 },
          "targetRef": { "type": "string", "minLength": 1 },
          "result": { "type": "string", "minLength": 1 },
          "sha256": { "type": "string", "pattern": "^[A-Fa-f0-9]{64}$" }
        }
      }
    }
  }
}
```

`contracts/fabric-human-gate.schema.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://opl.fabric/contracts/fabric-human-gate.schema.json",
  "title": "OPL Fabric Human Gate",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "gateId", "gateKind", "requestedAt", "status", "resourceRef", "decisionOptions"],
  "properties": {
    "version": { "const": "fabric-human-gate.v1" },
    "gateId": { "type": "string", "minLength": 1 },
    "gateKind": { "enum": ["destroy_storage", "delete_backup", "mutate_shared_ingress"] },
    "requestedAt": { "type": "string", "format": "date-time" },
    "status": { "enum": ["requested", "approved", "rejected", "changes_requested", "expired"] },
    "resourceRef": { "type": "string", "minLength": 1 },
    "evidenceRefs": { "type": "array", "items": { "type": "string", "minLength": 1 } },
    "decisionOptions": { "type": "array", "items": { "type": "string", "minLength": 1 } }
  }
}
```

`contracts/fabric-runtime-supervision.schema.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://opl.fabric/contracts/fabric-runtime-supervision.schema.json",
  "title": "OPL Fabric Runtime Supervision",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "provider", "ready", "observedAt", "blockers", "repairHints"],
  "properties": {
    "version": { "const": "fabric-runtime-supervision.v1" },
    "provider": { "type": "string", "minLength": 1 },
    "ready": { "type": "boolean" },
    "observedAt": { "type": "string", "format": "date-time" },
    "blockers": { "type": "array", "items": { "type": "string", "minLength": 1 } },
    "repairHints": { "type": "array", "items": { "type": "string", "minLength": 1 } }
  }
}
```

- [ ] **Step 5: Add OpenAPI skeleton**

Create `contracts/fabric-api.openapi.json`:

```json
{
  "openapi": "3.1.0",
  "info": {
    "title": "OPL Fabric API",
    "version": "0.1.0",
    "license": {
      "name": "Proprietary",
      "url": "https://github.com/RenDeHuang/OPL-Fabric"
    }
  },
  "servers": [
    {
      "url": "http://127.0.0.1:8787"
    }
  ],
  "security": [
    {
      "operatorToken": []
    }
  ],
  "components": {
    "securitySchemes": {
      "operatorToken": {
        "type": "http",
        "scheme": "bearer"
      }
    }
  },
  "paths": {
    "/api/fabric/readiness": {
      "get": {
        "operationId": "getFabricReadiness",
        "summary": "Get Fabric readiness",
        "responses": {
          "200": { "description": "Fabric readiness" },
          "400": { "description": "Bad request" }
        }
      }
    },
    "/api/fabric/catalog": {
      "get": {
        "operationId": "getFabricCatalog",
        "summary": "Get Fabric resource catalog",
        "responses": {
          "200": { "description": "Fabric resource catalog" },
          "400": { "description": "Bad request" }
        }
      }
    },
    "/api/fabric/compute": {
      "post": {
        "operationId": "createFabricCompute",
        "summary": "Create Fabric compute",
        "responses": {
          "202": { "description": "Compute operation accepted" },
          "400": { "description": "Bad request" }
        }
      }
    },
    "/api/fabric/storage": {
      "post": {
        "operationId": "createFabricStorage",
        "summary": "Create Fabric storage",
        "responses": {
          "202": { "description": "Storage operation accepted" },
          "400": { "description": "Bad request" }
        }
      }
    },
    "/api/fabric/attachments": {
      "post": {
        "operationId": "createFabricAttachment",
        "summary": "Create Fabric storage attachment",
        "responses": {
          "202": { "description": "Attachment operation accepted" },
          "400": { "description": "Bad request" }
        }
      }
    },
    "/api/fabric/workspace-routes": {
      "post": {
        "operationId": "createFabricWorkspaceRoute",
        "summary": "Create Fabric Workspace route",
        "responses": {
          "202": { "description": "Route operation accepted" },
          "400": { "description": "Bad request" }
        }
      }
    },
    "/api/fabric/operations/{operationId}": {
      "get": {
        "operationId": "getFabricOperation",
        "summary": "Get Fabric operation status",
        "parameters": [
          {
            "name": "operationId",
            "in": "path",
            "required": true,
            "schema": { "type": "string", "minLength": 1 }
          }
        ],
        "responses": {
          "200": { "description": "Operation status" },
          "400": { "description": "Bad request" }
        }
      }
    }
  }
}
```

- [ ] **Step 6: Verify contract tests pass**

Run:

```bash
npm run test:contracts
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add contracts tests/contracts
git commit -m "feat: add fabric contract schemas"
```

## Task 3: Go Domain Model And Catalog

**Files:**
- Create: `apps/fabric-api/go.mod`
- Create: `apps/fabric-api/internal/domain/types.go`
- Create: `apps/fabric-api/internal/domain/rules.go`
- Create: `apps/fabric-api/internal/domain/rules_test.go`
- Create: `apps/fabric-api/internal/catalog/catalog.go`
- Create: `apps/fabric-api/internal/catalog/catalog_test.go`
- Create: `apps/fabric-api/internal/config/config.go`

- [ ] **Step 1: Create Go module**

Write `apps/fabric-api/go.mod`:

```go
module github.com/RenDeHuang/OPL-Fabric/apps/fabric-api

go 1.22.2
```

Replace `go.work` with:

```go
go 1.22.2

use ./apps/fabric-api
```

- [ ] **Step 2: Write failing domain safety tests**

Create `apps/fabric-api/internal/domain/rules_test.go`:

```go
package domain

import "testing"

func TestDestroyStorageRequiresConfirmation(t *testing.T) {
	resource := StorageVolume{ID: "storage-1", State: StorageAvailable}

	tests := []struct {
		name    string
		storage StorageVolume
		req     DestroyStorageRequest
		wantErr error
	}{
		{
			name:    "missing_requested_by",
			storage: resource,
			req:     DestroyStorageRequest{Confirm: true},
			wantErr: ErrRequestedByRequired,
		},
		{
			name:    "missing_confirmation",
			storage: resource,
			req:     DestroyStorageRequest{RequestedBy: "operator"},
			wantErr: ErrStorageDestroyRequiresConfirmation,
		},
		{
			name:    "missing_storage_id",
			storage: StorageVolume{State: StorageAvailable},
			req:     DestroyStorageRequest{Confirm: true, RequestedBy: "operator"},
			wantErr: ErrStorageIDRequired,
		},
		{
			name:    "already_destroyed",
			storage: StorageVolume{ID: "storage-1", State: StorageDestroyed},
			req:     DestroyStorageRequest{Confirm: true, RequestedBy: "operator"},
			wantErr: ErrStorageAlreadyDestroyed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanDestroyStorage(tt.storage, tt.req)
			if err != tt.wantErr {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	err := CanDestroyStorage(resource, DestroyStorageRequest{Confirm: true, RequestedBy: "operator"})
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
```

- [ ] **Step 3: Run domain tests and confirm failure**

Run:

```bash
go test ./apps/fabric-api/internal/domain
```

Expected: FAIL because domain types and functions do not exist.

- [ ] **Step 4: Add domain types and rules**

Create `apps/fabric-api/internal/domain/types.go`:

```go
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
	Confirm      bool
	HumanGateID  string
	RequestedBy  string
}

type DestroyResult struct {
	Compute ComputeResource
	Storage StorageVolume
}
```

Create `apps/fabric-api/internal/domain/rules.go`:

```go
package domain

import "errors"

var (
	ErrStorageDestroyRequiresConfirmation = errors.New("storage_destroy_requires_confirmation")
	ErrRequestedByRequired                = errors.New("requested_by_required")
	ErrStorageIDRequired                  = errors.New("storage_id_required")
	ErrStorageAlreadyDestroyed            = errors.New("storage_already_destroyed")
)

func CanDestroyStorage(storage StorageVolume, req DestroyStorageRequest) error {
	if req.RequestedBy == "" {
		return ErrRequestedByRequired
	}
	if storage.ID == "" {
		return ErrStorageIDRequired
	}
	if storage.State == StorageDestroyed || storage.State == StorageDestroying {
		return ErrStorageAlreadyDestroyed
	}
	if !req.Confirm && req.HumanGateID == "" {
		return ErrStorageDestroyRequiresConfirmation
	}
	return nil
}

func DestroyCompute(compute ComputeResource, storage StorageVolume) (DestroyResult, error) {
	compute.State = ComputeDestroying
	return DestroyResult{Compute: compute, Storage: storage}, nil
}
```

- [ ] **Step 5: Add catalog tests**

Create `apps/fabric-api/internal/catalog/catalog_test.go`:

```go
package catalog

import "testing"

func TestDefaultCatalogPackages(t *testing.T) {
	catalog := DefaultCatalog(Config{
		WorkspaceImage: "ghcr.io/gaofeng21cn/one-person-lab-app:latest",
		WorkspaceDomain: "workspace.medopl.cn",
		StorageClass: "cbs",
	})

	if len(catalog.WorkspacePackages) != 3 {
		t.Fatalf("workspace package count = %d", len(catalog.WorkspacePackages))
	}

	basic := catalog.WorkspacePackages[0]
	if basic.ID != "basic" || !basic.Available || basic.CPU != 2 || basic.MemoryGB != 4 || basic.DiskGB != 10 {
		t.Fatalf("basic package mismatch: %+v", basic)
	}

	gpu := catalog.WorkspacePackages[2]
	if gpu.ID != "gpu" || gpu.Available || gpu.UnavailableReason != "gpu_node_pool_not_verified" {
		t.Fatalf("gpu package mismatch: %+v", gpu)
	}

	for _, profile := range catalog.ComputeProfiles {
		if profile.Accelerator == "" {
			t.Fatalf("compute profile %s missing accelerator", profile.ID)
		}
	}
}
```

- [ ] **Step 6: Add catalog implementation**

Create `apps/fabric-api/internal/catalog/catalog.go`:

```go
package catalog

type Config struct {
	WorkspaceImage  string
	WorkspaceDomain string
	StorageClass    string
}

type Catalog struct {
	SchemaVersion     int                `json:"schemaVersion"`
	Owner             string             `json:"owner"`
	WorkspacePackages []WorkspacePackage `json:"workspacePackages"`
	ComputeProfiles   []ComputeProfile   `json:"computeProfiles"`
	StorageClasses    []StorageClass     `json:"storageClasses"`
	WorkspaceImages   []WorkspaceImage   `json:"workspaceImages"`
	IngressDomains    []IngressDomain    `json:"ingressDomains"`
}

type WorkspacePackage struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Accelerator         string `json:"accelerator"`
	CPU                 int    `json:"cpu"`
	MemoryGB            int    `json:"memoryGb"`
	GPU                 int    `json:"gpu"`
	Server              string `json:"server"`
	DiskGB              int    `json:"diskGb"`
	Available           bool   `json:"available"`
	UnavailableReason   string `json:"unavailableReason,omitempty"`
	ComputeProfileID    string `json:"computeProfileId"`
	StorageClassID      string `json:"storageClassId"`
	WorkspaceImageID    string `json:"workspaceImageId"`
	IngressDomainID     string `json:"ingressDomainId"`
}

type ComputeProfile struct {
	ID          string `json:"id"`
	Accelerator string `json:"accelerator"`
	Provider    string `json:"provider"`
	CPU         int    `json:"cpu"`
	MemoryGB    int    `json:"memoryGb"`
	GPU         int    `json:"gpu"`
	Available   bool   `json:"available"`
}

type StorageClass struct {
	ID               string `json:"id"`
	Provider         string `json:"provider"`
	StorageClassName string `json:"storageClassName"`
	AccessMode       string `json:"accessMode"`
	Available        bool   `json:"available"`
}

type WorkspaceImage struct {
	ID               string   `json:"id"`
	Image            string   `json:"image"`
	Port             int      `json:"port"`
	PersistentMounts []string `json:"persistentMounts"`
	Available        bool     `json:"available"`
}

type IngressDomain struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	PathPattern string `json:"pathPattern"`
	Available   bool   `json:"available"`
}

func DefaultCatalog(cfg Config) Catalog {
	return Catalog{
		SchemaVersion: 1,
		Owner:         "OPL Fabric",
		WorkspacePackages: []WorkspacePackage{
			{ID: "basic", Name: "Basic Workspace", Accelerator: "cpu", CPU: 2, MemoryGB: 4, GPU: 0, Server: "2c4g", DiskGB: 10, Available: true, ComputeProfileID: "cpu-basic", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace"},
			{ID: "pro", Name: "Pro Workspace", Accelerator: "cpu", CPU: 8, MemoryGB: 16, GPU: 0, Server: "8c16g", DiskGB: 100, Available: true, ComputeProfileID: "cpu-pro", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace"},
			{ID: "gpu", Name: "GPU Workspace", Accelerator: "gpu", CPU: 16, MemoryGB: 64, GPU: 1, Server: "16c64g-1gpu", DiskGB: 500, Available: false, UnavailableReason: "gpu_node_pool_not_verified", ComputeProfileID: "gpu-standard", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace"},
		},
		ComputeProfiles: []ComputeProfile{
			{ID: "cpu-basic", Accelerator: "cpu", Provider: "tencent-tke", CPU: 2, MemoryGB: 4, GPU: 0, Available: true},
			{ID: "cpu-pro", Accelerator: "cpu", Provider: "tencent-tke", CPU: 8, MemoryGB: 16, GPU: 0, Available: true},
			{ID: "gpu-standard", Accelerator: "gpu", Provider: "tencent-tke", CPU: 16, MemoryGB: 64, GPU: 1, Available: false},
		},
		StorageClasses:  []StorageClass{{ID: "workspace-cbs", Provider: "tencent-tke", StorageClassName: cfg.StorageClass, AccessMode: "ReadWriteOnce", Available: true}},
		WorkspaceImages: []WorkspaceImage{{ID: "one-person-lab-app", Image: cfg.WorkspaceImage, Port: 3000, PersistentMounts: []string{"/data", "/projects"}, Available: true}},
		IngressDomains:  []IngressDomain{{ID: "workspace", Host: cfg.WorkspaceDomain, PathPattern: "/w/<workspaceId>", Available: true}},
	}
}
```

- [ ] **Step 7: Add config defaults**

Create `apps/fabric-api/internal/config/config.go`:

```go
package config

import "os"

type Config struct {
	Port             string
	DatabaseURL      string
	WorkspaceImage   string
	WorkspaceDomain  string
	StorageClass     string
	KubernetesNamespace string
}

func Load() Config {
	return Config{
		Port:                env("PORT", "8787"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		WorkspaceImage:      env("OPL_WORKSPACE_IMAGE", "ghcr.io/gaofeng21cn/one-person-lab-app:latest"),
		WorkspaceDomain:     env("OPL_WORKSPACE_DOMAIN", "workspace.medopl.cn"),
		StorageClass:        env("OPL_WORKSPACE_STORAGE_CLASS", "cbs"),
		KubernetesNamespace: env("OPL_K8S_NAMESPACE", "opl-cloud"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
```

- [ ] **Step 8: Run Go tests**

Run:

```bash
go test ./apps/fabric-api/internal/domain ./apps/fabric-api/internal/catalog
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add apps/fabric-api go.work
git commit -m "feat: add fabric domain and catalog"
```

## Task 4: Evidence And Operation Model

**Files:**
- Create: `apps/fabric-api/internal/evidence/evidence.go`
- Create: `apps/fabric-api/internal/evidence/evidence_test.go`
- Modify: `apps/fabric-api/internal/domain/types.go`

- [ ] **Step 1: Write failing evidence tests**

Create `apps/fabric-api/internal/evidence/evidence_test.go`:

```go
package evidence

import "testing"

func TestDigestManifestIsStable(t *testing.T) {
	input := []byte(`{"kind":"Deployment","metadata":{"name":"opl-ws"}}`)
	first := Digest(input)
	second := Digest(input)
	if first != second {
		t.Fatalf("digest changed: %s != %s", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("digest length = %d", len(first))
	}
	if got, want := Digest(nil), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"; got != want {
		t.Fatalf("empty digest = %s, want %s", got, want)
	}
	if got, want := Digest([]byte("manifest")), "05b3abf2579a5eb66403cd78be557fd860633a1fe2103c7642030defe32c657f"; got != want {
		t.Fatalf("manifest digest = %s, want %s", got, want)
	}
}

func TestLifecycleLedgerEntry(t *testing.T) {
	entry := LedgerEntry{
		OperationID: "op-1",
		Phase: "apply",
		Status: "succeeded",
		Summary: "applied deployment",
		Actions: []LedgerAction{{ActionID: "act-1", ActionKind: "apply", TargetRef: "deployment/opl-ws", Result: "created", SHA256: Digest([]byte("manifest"))}},
	}
	if err := entry.Validate(); err != nil {
		t.Fatalf("entry should validate: %v", err)
	}
}

func TestLifecycleLedgerEntryValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		entry LedgerEntry
		want  error
	}{
		{
			name:  "missing_operation_id",
			entry: LedgerEntry{Phase: "apply", Status: "succeeded", Summary: "summary", Actions: []LedgerAction{validAction()}},
			want:  ErrLedgerOperationIDRequired,
		},
		{
			name:  "missing_action",
			entry: LedgerEntry{OperationID: "op-1", Phase: "apply", Status: "succeeded", Summary: "summary"},
			want:  ErrLedgerActionRequired,
		},
		{
			name:  "missing_action_target",
			entry: LedgerEntry{OperationID: "op-1", Phase: "apply", Status: "succeeded", Summary: "summary", Actions: []LedgerAction{{ActionID: "act-1", ActionKind: "apply", Result: "created", SHA256: Digest([]byte("manifest"))}}},
			want:  ErrLedgerActionTargetRequired,
		},
		{
			name:  "invalid_sha",
			entry: LedgerEntry{OperationID: "op-1", Phase: "apply", Status: "succeeded", Summary: "summary", Actions: []LedgerAction{{ActionID: "act-1", ActionKind: "apply", TargetRef: "deployment/opl-ws", Result: "created", SHA256: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"}}},
			want:  ErrLedgerActionSHAInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.entry.Validate(); err != tt.want {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func validAction() LedgerAction {
	return LedgerAction{
		ActionID:   "act-1",
		ActionKind: "apply",
		TargetRef:  "deployment/opl-ws",
		Result:     "created",
		SHA256:     Digest([]byte("manifest")),
	}
}
```

- [ ] **Step 2: Run evidence tests and confirm failure**

Run:

```bash
go test ./apps/fabric-api/internal/evidence
```

Expected: FAIL because evidence package is not implemented.

- [ ] **Step 3: Add evidence implementation**

Create `apps/fabric-api/internal/evidence/evidence.go`:

```go
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	ErrLedgerOperationIDRequired   = errors.New("ledger_operation_id_required")
	ErrLedgerPhaseRequired         = errors.New("ledger_phase_required")
	ErrLedgerStatusRequired        = errors.New("ledger_status_required")
	ErrLedgerSummaryRequired       = errors.New("ledger_summary_required")
	ErrLedgerActionRequired        = errors.New("ledger_entry_requires_action")
	ErrLedgerActionIDRequired      = errors.New("ledger_action_id_required")
	ErrLedgerActionKindRequired    = errors.New("ledger_action_kind_required")
	ErrLedgerActionTargetRequired  = errors.New("ledger_action_target_required")
	ErrLedgerActionResultRequired  = errors.New("ledger_action_result_required")
	ErrLedgerActionSHAInvalid      = errors.New("ledger_action_sha_invalid")
)

type LedgerEntry struct {
	OperationID string         `json:"operationId"`
	Phase       string         `json:"phase"`
	Status      string         `json:"status"`
	Summary     string         `json:"summary"`
	Actions     []LedgerAction `json:"actions"`
}

type LedgerAction struct {
	ActionID   string `json:"actionId"`
	ActionKind string `json:"actionKind"`
	TargetRef  string `json:"targetRef"`
	Result     string `json:"result"`
	SHA256     string `json:"sha256"`
}

func Digest(input []byte) string {
	sum := sha256.Sum256(input)
	return hex.EncodeToString(sum[:])
}

func (entry LedgerEntry) Validate() error {
	if entry.OperationID == "" {
		return ErrLedgerOperationIDRequired
	}
	if entry.Phase == "" {
		return ErrLedgerPhaseRequired
	}
	if entry.Status == "" {
		return ErrLedgerStatusRequired
	}
	if entry.Summary == "" {
		return ErrLedgerSummaryRequired
	}
	if len(entry.Actions) == 0 {
		return ErrLedgerActionRequired
	}
	for _, action := range entry.Actions {
		if action.ActionID == "" {
			return ErrLedgerActionIDRequired
		}
		if action.ActionKind == "" {
			return ErrLedgerActionKindRequired
		}
		if action.TargetRef == "" {
			return ErrLedgerActionTargetRequired
		}
		if action.Result == "" {
			return ErrLedgerActionResultRequired
		}
		if !isSHA256Hex(action.SHA256) {
			return ErrLedgerActionSHAInvalid
		}
	}
	return nil
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
```

- [ ] **Step 4: Extend domain operation types**

Append to `apps/fabric-api/internal/domain/types.go`:

```go
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
```

- [ ] **Step 5: Run evidence and domain tests**

Run:

```bash
go test ./apps/fabric-api/internal/evidence ./apps/fabric-api/internal/domain
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/fabric-api/internal/evidence apps/fabric-api/internal/domain/types.go
git commit -m "feat: add fabric evidence model"
```

## Task 5: PostgreSQL Schema And Store Boundary

**Files:**
- Create: `apps/fabric-api/internal/postgres/schema.sql`
- Create: `apps/fabric-api/internal/postgres/migrations.go`
- Create: `apps/fabric-api/internal/postgres/store.go`
- Create: `apps/fabric-api/internal/postgres/store_test.go`
- Modify: `apps/fabric-api/go.mod`

- [ ] **Step 1: Add store tests first**

Create `apps/fabric-api/internal/postgres/store_test.go`:

```go
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
```

- [ ] **Step 2: Run store tests and confirm failure**

Run:

```bash
go test ./apps/fabric-api/internal/postgres
```

Expected: FAIL because postgres package is not implemented.

- [ ] **Step 3: Add PostgreSQL dependency**

Modify `apps/fabric-api/go.mod` so `require` includes:

```go
	github.com/jackc/pgx/v5 v5.5.5
```

- [ ] **Step 4: Add schema**

Create `apps/fabric-api/internal/postgres/schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS compute_resources (
  id TEXT PRIMARY KEY,
  owner_account_id TEXT NOT NULL,
  package_id TEXT NOT NULL,
  state TEXT NOT NULL,
  provider_ref TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS storage_volumes (
  id TEXT PRIMARY KEY,
  owner_account_id TEXT NOT NULL,
  package_id TEXT NOT NULL,
  state TEXT NOT NULL,
  provider_ref TEXT NOT NULL DEFAULT '',
  size_gb INTEGER NOT NULL CHECK (size_gb > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS storage_attachments (
  id TEXT PRIMARY KEY,
  compute_id TEXT NOT NULL REFERENCES compute_resources(id),
  storage_id TEXT NOT NULL REFERENCES storage_volumes(id),
  state TEXT NOT NULL,
  mount_path TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workspace_routes (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  compute_id TEXT NOT NULL REFERENCES compute_resources(id),
  state TEXT NOT NULL,
  host TEXT NOT NULL,
  path TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS storage_backups (
  id TEXT PRIMARY KEY,
  storage_id TEXT NOT NULL REFERENCES storage_volumes(id),
  state TEXT NOT NULL,
  provider_ref TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fabric_operations (
  id TEXT PRIMARY KEY,
  correlation_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  requested_by TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  resource_kind TEXT NOT NULL,
  state TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fabric_events (
  id TEXT PRIMARY KEY,
  operation_id TEXT NOT NULL REFERENCES fabric_operations(id),
  event_name TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fabric_evidence_refs (
  id TEXT PRIMARY KEY,
  operation_id TEXT NOT NULL REFERENCES fabric_operations(id),
  kind TEXT NOT NULL,
  ref TEXT NOT NULL,
  sha256 TEXT NOT NULL CHECK (sha256 ~ '^[A-Fa-f0-9]{64}$'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS human_gates (
  id TEXT PRIMARY KEY,
  gate_kind TEXT NOT NULL,
  resource_ref TEXT NOT NULL,
  status TEXT NOT NULL,
  requested_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
  key TEXT PRIMARY KEY,
  operation_id TEXT NOT NULL UNIQUE REFERENCES fabric_operations(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Create `apps/fabric-api/internal/postgres/migrations.go`:

```go
package postgres

import _ "embed"

//go:embed schema.sql
var SchemaSQL string
```

- [ ] **Step 5: Add store interface**

Create `apps/fabric-api/internal/postgres/store.go`:

```go
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
	_, err := s.pool.Exec(ctx, SchemaSQL)
	return err
}
```

- [ ] **Step 6: Run postgres package tests**

Run:

```bash
go mod tidy -C apps/fabric-api
go test ./apps/fabric-api/internal/postgres
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/fabric-api/go.mod apps/fabric-api/go.sum apps/fabric-api/internal/postgres
git commit -m "feat: add postgres schema boundary"
```

## Task 6: Kubernetes Provider Boundary

**Files:**
- Create: `apps/fabric-api/internal/k8s/provider.go`
- Create: `apps/fabric-api/internal/k8s/provider_test.go`

- [ ] **Step 1: Write failing fake-client provider test**

Create `apps/fabric-api/internal/k8s/provider_test.go`:

```go
package k8s

import (
	"context"
	"errors"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	testingclient "k8s.io/client-go/testing"
)

func TestCreateComputeCreatesDeploymentAndService(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            "compute-1",
		WorkspaceName: "Alpha",
		PackageID:     "basic",
		CPU:           2,
		MemoryGB:      4,
	})
	if err != nil {
		t.Fatalf("create compute failed: %v", err)
	}
	if !strings.HasPrefix(result.ProviderRef, "deployment/opl-compute-1-") {
		t.Fatalf("provider ref = %s", result.ProviderRef)
	}
	if !strings.HasPrefix(result.ServiceRef, "service/opl-compute-1-") {
		t.Fatalf("service ref = %s", result.ServiceRef)
	}
	name := strings.TrimPrefix(result.ProviderRef, "deployment/")

	deploy, err := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	if deploy.Spec.Template.Spec.Containers[0].Image != "workspace-image:latest" {
		t.Fatalf("image mismatch")
	}
	if deploy.Annotations["oplcloud.cn/compute-id"] != "compute-1" {
		t.Fatalf("raw compute id annotation missing")
	}
	if deploy.Spec.Template.Spec.AutomountServiceAccountToken == nil || *deploy.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatalf("automount service account token should be false")
	}
	if deploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != 3000 {
		t.Fatalf("container port mismatch")
	}
	if deploy.Spec.Selector.MatchLabels["oplcloud.cn/compute-key"] == "" {
		t.Fatalf("selector missing label-safe compute key")
	}

	service, err := client.CoreV1().Services("opl-cloud").Get(context.Background(), strings.TrimPrefix(result.ServiceRef, "service/"), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("service missing: %v", err)
	}
	if service.Spec.Selector["oplcloud.cn/compute-key"] != deploy.Spec.Template.Labels["oplcloud.cn/compute-key"] {
		t.Fatalf("service selector does not match deployment label")
	}
	if service.Spec.Ports[0].Port != 3000 {
		t.Fatalf("service port mismatch")
	}
}

func TestCreateComputeUsesBoundedDNSNameAndSafeLabels(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}
	longID := "Compute_" + strings.Repeat("ABC123_", 20)

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            longID,
		WorkspaceName: "Alpha",
		PackageID:     "basic",
	})
	if err != nil {
		t.Fatalf("create compute failed: %v", err)
	}
	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	if len(name) > 63 {
		t.Fatalf("name length = %d", len(name))
	}
	if strings.Contains(name, "_") {
		t.Fatalf("name contains unsafe character: %s", name)
	}
	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		t.Fatalf("name is not a DNS-1123 label: %v", errs)
	}

	deploy, err := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	if deploy.Annotations["oplcloud.cn/compute-id"] != longID {
		t.Fatalf("raw compute id annotation mismatch")
	}
	for key, value := range deploy.Labels {
		if value == longID {
			t.Fatalf("raw compute id leaked into label %s", key)
		}
		if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
			t.Fatalf("label %s value %q is invalid: %v", key, value, errs)
		}
	}
}

func TestCreateComputeCleansDeploymentWhenServiceCreateFails(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "services", func(action testingclient.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("service_create_failed")
	})
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}

	_, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            "compute-1",
		WorkspaceName: "Alpha",
		PackageID:     "basic",
	})
	if err == nil {
		t.Fatal("expected service create failure")
	}
	name := strings.TrimPrefix(k8sName("compute-1"), "deployment/")
	_, getErr := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), name, metav1.GetOptions{})
	if getErr == nil {
		t.Fatal("deployment should be cleaned up after service failure")
	}
}

func TestCreateComputeReportsCleanupFailure(t *testing.T) {
	client := fake.NewSimpleClientset()
	serviceErr := errors.New("service_create_failed")
	deleteErr := errors.New("deployment_delete_failed")
	client.PrependReactor("create", "services", func(action testingclient.Action) (bool, runtime.Object, error) {
		return true, nil, serviceErr
	})
	client.PrependReactor("delete", "deployments", func(action testingclient.Action) (bool, runtime.Object, error) {
		return true, nil, deleteErr
	})
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}

	_, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            "compute-1",
		WorkspaceName: "Alpha",
		PackageID:     "basic",
	})
	if !errors.Is(err, serviceErr) {
		t.Fatalf("expected service error, got %v", err)
	}
	if !errors.Is(err, deleteErr) {
		t.Fatalf("expected cleanup error, got %v", err)
	}
}

var _ = appsv1.Deployment{}
var _ = corev1.Service{}
var _ kubernetes.Interface = fake.NewSimpleClientset()
```

- [ ] **Step 2: Run provider test and confirm failure**

Run:

```bash
go test ./apps/fabric-api/internal/k8s
```

Expected: FAIL because provider code is not implemented.

- [ ] **Step 3: Add provider implementation**

Create `apps/fabric-api/internal/k8s/provider.go`:

```go
package k8s

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type Provider struct {
	Client         kubernetes.Interface
	Namespace      string
	WorkspaceImage string
}

type CreateComputeInput struct {
	ID            string
	WorkspaceName string
	PackageID     string
	CPU           int
	MemoryGB      int
}

type CreateComputeResult struct {
	ProviderRef string
	ServiceRef  string
}

func (p Provider) CreateCompute(ctx context.Context, input CreateComputeInput) (CreateComputeResult, error) {
	name := k8sName(input.ID)
	computeKey := labelValue(input.ID)
	labels := map[string]string{
		"app.kubernetes.io/name":     "opl-workspace",
		"app.kubernetes.io/instance": name,
		"oplcloud.cn/compute-key":    computeKey,
	}
	annotations := map[string]string{
		"oplcloud.cn/compute-id": input.ID,
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.Namespace, Labels: labels, Annotations: annotations},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr[int32](1),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr(false),
					Containers: []corev1.Container{{
						Name:  "workspace",
						Image: p.WorkspaceImage,
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 3000}},
						Env: []corev1.EnvVar{
							{Name: "OPL_COMPUTE_ID", Value: input.ID},
							{Name: "OPL_WORKSPACE_NAME", Value: input.WorkspaceName},
							{Name: "OPL_PACKAGE_ID", Value: input.PackageID},
						},
					}},
				},
			},
		},
	}
	if _, err := p.Client.AppsV1().Deployments(p.Namespace).Create(ctx, deploy, metav1.CreateOptions{}); err != nil {
		return CreateComputeResult{}, err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.Namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Name: "http", Port: 3000, TargetPort: intstr.FromInt(3000)}},
		},
	}
	if _, err := p.Client.CoreV1().Services(p.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		if deleteErr := p.Client.AppsV1().Deployments(p.Namespace).Delete(ctx, name, metav1.DeleteOptions{}); deleteErr != nil {
			return CreateComputeResult{}, errors.Join(err, fmt.Errorf("cleanup deployment %q: %w", name, deleteErr))
		}
		return CreateComputeResult{}, err
	}

	return CreateComputeResult{ProviderRef: "deployment/" + name, ServiceRef: "service/" + name}, nil
}

func k8sName(id string) string {
	return boundedName("opl", id, 63)
}

func labelValue(id string) string {
	return boundedName("compute", id, 63)
}

func boundedName(prefix, id string, limit int) string {
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return '-'
	}, id)
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "resource"
	}
	hash := shortHash(id)
	maxClean := limit - len(prefix) - len(hash) - 2
	if maxClean < 1 {
		maxClean = 1
	}
	if len(clean) > maxClean {
		clean = strings.Trim(clean[:maxClean], "-")
	}
	if clean == "" {
		clean = "resource"
	}
	return fmt.Sprintf("%s-%s-%s", prefix, clean, hash)
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func ptr[T any](value T) *T {
	return &value
}
```

- [ ] **Step 4: Run provider tests**

Run:

```bash
go test ./apps/fabric-api/internal/k8s
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/fabric-api/internal/k8s
git commit -m "feat: add client-go provider boundary"
```

## Task 7: Service And HTTP API

**Files:**
- Create: `apps/fabric-api/internal/service/service.go`
- Create: `apps/fabric-api/internal/http/server.go`
- Create: `apps/fabric-api/internal/http/server_test.go`
- Create: `apps/fabric-api/cmd/fabric-api/main.go`

- [ ] **Step 1: Write failing HTTP tests**

Create `apps/fabric-api/internal/http/server_test.go`:

```go
package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
)

func TestReadinessEndpoint(t *testing.T) {
	svc := service.New(service.Config{
		Catalog: catalog.DefaultCatalog(catalog.Config{WorkspaceImage: "image", WorkspaceDomain: "workspace.medopl.cn", StorageClass: "cbs"}),
	})
	server := NewServer(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/fabric/readiness", nil)
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["provider"] != "tencent-tke" {
		t.Fatalf("provider = %v", body["provider"])
	}
}

func TestCatalogEndpoint(t *testing.T) {
	svc := service.New(service.Config{
		Catalog: catalog.DefaultCatalog(catalog.Config{WorkspaceImage: "image", WorkspaceDomain: "workspace.medopl.cn", StorageClass: "cbs"}),
	})
	server := NewServer(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/fabric/catalog", nil)
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run HTTP tests and confirm failure**

Run:

```bash
go test ./apps/fabric-api/internal/http
```

Expected: FAIL because service and server packages do not exist.

- [ ] **Step 3: Add service implementation**

Create `apps/fabric-api/internal/service/service.go`:

```go
package service

import "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"

type Config struct {
	Catalog catalog.Catalog
}

type Service struct {
	catalog catalog.Catalog
}

type Readiness struct {
	Ready        bool             `json:"ready"`
	Provider     string           `json:"provider"`
	MissingEnv   []string         `json:"missingEnv"`
	ResourceCatalog catalog.Catalog `json:"resourceCatalog"`
	Blockers     []string         `json:"blockers"`
	RepairHints  []string         `json:"repairHints"`
}

func New(cfg Config) *Service {
	return &Service{catalog: cfg.Catalog}
}

func (s *Service) Catalog() catalog.Catalog {
	return s.catalog
}

func (s *Service) Readiness() Readiness {
	return Readiness{
		Ready: true,
		Provider: "tencent-tke",
		MissingEnv: []string{},
		ResourceCatalog: s.catalog,
		Blockers: []string{},
		RepairHints: []string{},
	}
}
```

- [ ] **Step 4: Add HTTP server**

Create `apps/fabric-api/internal/http/server.go`:

```go
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
)

func NewServer(svc *service.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/fabric/readiness", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, svc.Readiness())
	})
	mux.HandleFunc("GET /api/fabric/catalog", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, svc.Catalog())
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
```

- [ ] **Step 5: Add main**

Create `apps/fabric-api/cmd/fabric-api/main.go`:

```go
package main

import (
	"log"
	"net/http"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/config"
	httpapi "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/http"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
)

func main() {
	cfg := config.Load()
	cat := catalog.DefaultCatalog(catalog.Config{
		WorkspaceImage: cfg.WorkspaceImage,
		WorkspaceDomain: cfg.WorkspaceDomain,
		StorageClass: cfg.StorageClass,
	})
	svc := service.New(service.Config{Catalog: cat})
	server := httpapi.NewServer(svc)

	addr := ":" + cfg.Port
	log.Printf("opl fabric api listening on %s", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 6: Run API tests**

Run:

```bash
go test ./apps/fabric-api/...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/fabric-api/internal/service apps/fabric-api/internal/http apps/fabric-api/cmd
git commit -m "feat: expose fabric readiness api"
```

## Task 8: React TypeScript Operator Console

**Files:**
- Create: `apps/fabric-console/package.json`
- Create: `apps/fabric-console/index.html`
- Create: `apps/fabric-console/tsconfig.json`
- Create: `apps/fabric-console/vite.config.ts`
- Create: `apps/fabric-console/src/api.ts`
- Create: `apps/fabric-console/src/App.tsx`
- Create: `apps/fabric-console/src/main.tsx`
- Create: `apps/fabric-console/src/styles.css`

- [ ] **Step 1: Add console package**

Create `apps/fabric-console/package.json`:

```json
{
  "name": "@opl/fabric-console",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite --host 127.0.0.1",
    "typecheck": "tsc --noEmit",
    "build": "tsc --noEmit && vite build"
  },
  "dependencies": {
    "@vitejs/plugin-react": "^4.3.4",
    "vite": "^6.0.7",
    "typescript": "^5.7.2",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "lucide-react": "^0.468.0"
  },
  "devDependencies": {}
}
```

- [ ] **Step 2: Add TypeScript and Vite config**

Create `apps/fabric-console/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["DOM", "DOM.Iterable", "ES2022"],
    "allowJs": false,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "strict": true,
    "forceConsistentCasingInFileNames": true,
    "module": "ESNext",
    "moduleResolution": "Node",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx"
  },
  "include": ["src"]
}
```

Create `apps/fabric-console/vite.config.ts`:

```ts
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://127.0.0.1:8787"
    }
  }
});
```

Create `apps/fabric-console/index.html`:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>OPL Fabric Operator Console</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 3: Add API client**

Create `apps/fabric-console/src/api.ts`:

```ts
export interface Readiness {
  ready: boolean;
  provider: string;
  missingEnv: string[];
  blockers: string[];
  repairHints: string[];
  resourceCatalog: Catalog;
}

export interface Catalog {
  workspacePackages: WorkspacePackage[];
}

export interface WorkspacePackage {
  id: string;
  name: string;
  accelerator: "cpu" | "gpu";
  cpu: number;
  memoryGb: number;
  gpu: number;
  diskGb: number;
  available: boolean;
  unavailableReason?: string;
}

export async function fetchReadiness(): Promise<Readiness> {
  const response = await fetch("/api/fabric/readiness");
  if (!response.ok) {
    throw new Error(`readiness_failed:${response.status}`);
  }
  return response.json() as Promise<Readiness>;
}
```

- [ ] **Step 4: Add operator UI**

Create `apps/fabric-console/src/App.tsx`:

```tsx
import { AlertTriangle, CheckCircle2, ServerCog } from "lucide-react";
import { useEffect, useState } from "react";
import { fetchReadiness, type Readiness } from "./api";
import "./styles.css";

export function App() {
  const [readiness, setReadiness] = useState<Readiness | null>(null);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    fetchReadiness()
      .then(setReadiness)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "readiness_unknown_error"));
  }, []);

  return (
    <main className="shell">
      <header className="topbar">
        <div className="brand">
          <ServerCog size={24} aria-hidden="true" />
          <div>
            <h1>OPL Fabric</h1>
            <p>Operator control plane</p>
          </div>
        </div>
        <div className={readiness?.ready ? "status ready" : "status blocked"}>
          {readiness?.ready ? <CheckCircle2 size={18} /> : <AlertTriangle size={18} />}
          <span>{readiness?.ready ? "Ready" : "Needs attention"}</span>
        </div>
      </header>

      {error ? <section className="notice">{error}</section> : null}

      <section className="panel">
        <h2>Runtime Readiness</h2>
        <dl className="facts">
          <div><dt>Provider</dt><dd>{readiness?.provider ?? "loading"}</dd></div>
          <div><dt>Missing env</dt><dd>{readiness?.missingEnv.length ?? 0}</dd></div>
          <div><dt>Blockers</dt><dd>{readiness?.blockers.length ?? 0}</dd></div>
        </dl>
      </section>

      <section className="panel">
        <h2>Workspace Packages</h2>
        <div className="grid">
          {(readiness?.resourceCatalog.workspacePackages ?? []).map((item) => (
            <article className="package" key={item.id}>
              <div className="packageHeader">
                <h3>{item.name}</h3>
                <span className={item.available ? "pill available" : "pill unavailable"}>
                  {item.available ? "available" : "blocked"}
                </span>
              </div>
              <p>{item.cpu} CPU / {item.memoryGb}GB memory / {item.diskGb}GB storage</p>
              {!item.available && item.unavailableReason ? <p className="muted">{item.unavailableReason}</p> : null}
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
```

Create `apps/fabric-console/src/main.tsx`:

```tsx
import { createRoot } from "react-dom/client";
import { App } from "./App";

createRoot(document.getElementById("root") as HTMLElement).render(<App />);
```

Create `apps/fabric-console/src/styles.css`:

```css
:root {
  color: #172026;
  background: #f4f7f8;
  font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  letter-spacing: 0;
}

body {
  margin: 0;
}

.shell {
  min-height: 100vh;
  padding: 24px;
}

.topbar {
  align-items: center;
  display: flex;
  justify-content: space-between;
  margin-bottom: 24px;
}

.brand {
  align-items: center;
  display: flex;
  gap: 12px;
}

h1, h2, h3, p {
  margin: 0;
}

h1 {
  font-size: 22px;
}

h2 {
  font-size: 16px;
  margin-bottom: 16px;
}

.brand p, .muted {
  color: #5d6972;
  font-size: 13px;
}

.status, .pill {
  align-items: center;
  border-radius: 6px;
  display: inline-flex;
  font-size: 13px;
  gap: 6px;
  padding: 6px 10px;
}

.ready, .available {
  background: #e4f6ee;
  color: #12623d;
}

.blocked, .unavailable {
  background: #fff1dd;
  color: #8a4b00;
}

.notice {
  background: #ffe5e5;
  border: 1px solid #f0b4b4;
  border-radius: 6px;
  margin-bottom: 16px;
  padding: 12px;
}

.panel {
  background: #ffffff;
  border: 1px solid #d9e1e5;
  border-radius: 8px;
  margin-bottom: 16px;
  padding: 16px;
}

.facts {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  margin: 0;
}

.facts div {
  border-left: 3px solid #2d7f8f;
  padding-left: 10px;
}

dt {
  color: #5d6972;
  font-size: 12px;
}

dd {
  font-size: 18px;
  font-weight: 700;
  margin: 2px 0 0;
}

.grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.package {
  border: 1px solid #d9e1e5;
  border-radius: 8px;
  padding: 14px;
}

.packageHeader {
  align-items: center;
  display: flex;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}
```

- [ ] **Step 5: Install and verify console**

Run:

```bash
npm --prefix apps/fabric-console install
npm --prefix apps/fabric-console run typecheck
npm --prefix apps/fabric-console run build
```

Expected: all commands exit `0`.

- [ ] **Step 6: Commit**

```bash
git add apps/fabric-console
git commit -m "feat: add fabric operator console"
```

## Task 9: Deployment Manifests And Final Verification

**Files:**
- Create: `deploy/k8s/opl-fabric-api.yaml`
- Create: `deploy/migrations/README.md`
- Modify: `README.md`

- [ ] **Step 1: Add deployment manifest**

Create `deploy/k8s/opl-fabric-api.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opl-fabric-api
  labels:
    app.kubernetes.io/name: opl-fabric-api
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: opl-fabric-api
  template:
    metadata:
      labels:
        app.kubernetes.io/name: opl-fabric-api
    spec:
      serviceAccountName: opl-fabric-api
      containers:
        - name: api
          image: opl-fabric-api:local
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 8787
          env:
            - name: PORT
              value: "8787"
            - name: OPL_K8S_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
---
apiVersion: v1
kind: Service
metadata:
  name: opl-fabric-api
spec:
  selector:
    app.kubernetes.io/name: opl-fabric-api
  ports:
    - name: http
      port: 8787
      targetPort: 8787
```

Create `deploy/migrations/README.md`:

```markdown
# Migrations

The first schema is embedded in `apps/fabric-api/internal/postgres/schema.sql`.

Production rollout must run `Store.Migrate` before serving mutating Fabric API traffic.
```

- [ ] **Step 2: Update README verification section**

Append to `README.md`:

````markdown
## Development

Run backend tests:

```bash
go test ./apps/fabric-api/...
```

Run operator console checks:

```bash
npm --prefix apps/fabric-console run typecheck
npm --prefix apps/fabric-console run build
```
````

- [ ] **Step 3: Run full verification**

Run:

```bash
npm test
git diff --check
```

Expected: both commands exit `0`.

- [ ] **Step 4: Commit**

```bash
git add README.md deploy
git commit -m "chore: add fabric deployment skeleton"
```

## Self-Review Checklist

- Spec coverage:
  - Contract-first files are created in Task 2.
  - Go domain and safety rules are created in Task 3.
  - PostgreSQL schema is created in Task 5.
  - client-go provider boundary is created in Task 6.
  - HTTP readiness and catalog API are created in Task 7.
  - React TypeScript operator console is created in Task 8.
  - Deployment skeleton and verification are created in Task 9.
- Boundaries:
  - No Console commercial UI is implemented.
  - No billing ledger or wallet behavior is implemented.
  - No Gateway behavior is implemented.
  - No `kubectl` subprocess path is introduced.
- Required verification:
  - `npm run test:contracts`
  - `go test ./apps/fabric-api/...`
  - `npm --prefix apps/fabric-console run typecheck`
  - `npm --prefix apps/fabric-console run build`
  - `npm test`
  - `git diff --check`
