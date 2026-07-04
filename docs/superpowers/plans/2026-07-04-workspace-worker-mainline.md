# Workspace Worker Mainline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `POST /api/fabric/workspaces` the production-facing mainline that a worker fulfills by executing storage -> compute -> attach -> entry, while keeping single-resource APIs usable as lower-level operational/debug APIs.

**Architecture:** HTTP handlers only accept intent and persist durable rows. A background worker leases accepted operations, calls the orchestrator, and records operation state. The orchestrator treats `workspace` as an aggregate operation and applies its linked storage, compute, attachment, and workspace entry in order through the existing runtime interface.

**Tech Stack:** Go, PostgreSQL, OpenAPI + JSON Schema, Kubernetes client-go provider boundary, existing Fabric orchestrator/runtime/store packages.

---

### Task 1: Store Worker Lease Boundary

**Files:**
- Modify: `apps/fabric-api/internal/postgres/schema.sql`
- Modify: `apps/fabric-api/internal/postgres/store.go`
- Modify: `apps/fabric-api/internal/postgres/store_test.go`

- [x] Add operation worker columns to `fabric_operations`.

```sql
lease_owner TEXT NOT NULL DEFAULT '',
lease_expires_at TIMESTAMPTZ,
attempts INTEGER NOT NULL DEFAULT 0,
last_error TEXT NOT NULL DEFAULT '',
provider_refs JSONB NOT NULL DEFAULT '{}'::jsonb,
evidence_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
```

- [x] Add `ListAcceptedOperations(ctx, limit int)` to return accepted operations ordered by `created_at`.

```go
func (s *Store) ListAcceptedOperations(ctx context.Context, limit int) ([]OperationRow, error)
```

- [x] Add `LeaseOperation(ctx, id, owner string, ttl time.Duration)` to atomically claim one accepted operation.

```go
func (s *Store) LeaseOperation(ctx context.Context, id, owner string, ttl time.Duration) (bool, error)
```

- [x] Add `RecordOperationFailure(ctx, id string, err error)` to increment attempts, store `last_error`, and set state to `accepted` while `attempts < 3`; set state to `failed` when `attempts >= 3`.

```go
func (s *Store) RecordOperationFailure(ctx context.Context, id string, cause error) error
```

- [x] Extend nil-store tests for all new methods.

Run:

```bash
cd apps/fabric-api && go test ./internal/postgres
```

Expected: PASS.

### Task 2: Workspace Aggregate Orchestrator

**Files:**
- Modify: `apps/fabric-api/internal/orchestrator/orchestrator.go`
- Modify: `apps/fabric-api/internal/orchestrator/orchestrator_test.go`
- Modify: `apps/fabric-api/internal/postgres/store.go`

- [x] Extend orchestrator store interface with workspace get/update methods.

```go
GetWorkspace(context.Context, string) (postgres.WorkspaceRow, error)
UpdateWorkspace(context.Context, postgres.WorkspaceRow) error
```

- [x] Add `case "workspace"` in `apply`.

```go
case "workspace":
    return o.applyWorkspace(ctx, op.ResourceID)
```

- [x] Implement `applyWorkspace` as a strict ordered chain:

```text
workspace.state = provisioning
apply storage volume
apply compute resource
copy compute/storage refs into attachment provider input before attaching
apply storage attachment
apply workspace entry
workspace.state = running
```

- [x] Add tests proving successful workspace operation marks:

```text
storage.state = available
compute.state = running
attachment.state = attached
entry.state = ready
workspace.state = running
operation.state = succeeded
```

- [x] Add a failure test proving compute failure marks operation failed and leaves storage retained/available.

Run:

```bash
cd apps/fabric-api && go test ./internal/orchestrator
```

Expected: PASS.

### Task 3: Background Worker

**Files:**
- Create: `apps/fabric-api/internal/worker/worker.go`
- Create: `apps/fabric-api/internal/worker/worker_test.go`

- [x] Implement a worker that scans accepted operations.

```go
type Store interface {
    ListAcceptedOperations(context.Context, int) ([]postgres.OperationRow, error)
    LeaseOperation(context.Context, string, string, time.Duration) (bool, error)
    RecordOperationFailure(context.Context, string, error) error
}

type Orchestrator interface {
    Apply(context.Context, string) (orchestrator.Receipt, error)
}
```

- [x] Implement `RunOnce(ctx)`:

```text
list accepted operations
for each op:
  lease op
  if lease acquired, call orchestrator.Apply(ctx, op.ID)
  if apply fails, call RecordOperationFailure
```

- [x] Implement `Run(ctx)` loop with configurable interval, limit, owner, and lease TTL.

- [x] Add fake-store tests for:

```text
accepted workspace operation is leased and applied
unleased operation is skipped
failed apply records failure
multiple accepted single-resource operations are also processed
```

Run:

```bash
cd apps/fabric-api && go test ./internal/worker
```

Expected: PASS.

### Task 4: API Process Wiring Without Rollout

**Files:**
- Modify: `apps/fabric-api/internal/config/config.go`
- Modify: `apps/fabric-api/internal/config/config_test.go`
- Modify: `apps/fabric-api/cmd/fabric-api/main.go`
- Modify: `config/fabric.env.example`
- Modify: `deploy/k8s/opl-fabric-api.yaml`

- [x] Add worker config:

```text
OPL_FABRIC_WORKER_ENABLED=false
OPL_FABRIC_WORKER_OWNER=fabric-api
OPL_FABRIC_WORKER_INTERVAL=5s
OPL_FABRIC_WORKER_LEASE_TTL=60s
OPL_FABRIC_WORKER_BATCH_SIZE=10
```

- [x] Keep worker disabled by default in examples and deployment skeleton.

- [x] When enabled, initialize:

```text
postgres store
k8s client-go runtime provider
orchestrator
worker loop
```

- [x] Do not add any deployment rollout step.

Run:

```bash
cd apps/fabric-api && go test ./cmd/fabric-api ./internal/config
```

Expected: PASS.

### Task 5: Single-Resource Status APIs

**Files:**
- Modify: `contracts/fabric-api.openapi.json`
- Create: `contracts/fabric-storage-volume.schema.json`
- Create: `contracts/fabric-compute-resource.schema.json`
- Create: `contracts/fabric-storage-attachment.schema.json`
- Create: `contracts/fabric-workspace-entry.schema.json`
- Modify: `tests/contracts/contracts.test.mjs`
- Modify: `apps/fabric-api/internal/service/service.go`
- Modify: `apps/fabric-api/internal/http/server.go`
- Modify: `apps/fabric-api/internal/http/server_test.go`

- [x] Add public GET routes:

```text
GET /api/fabric/storage-volumes/{id}
GET /api/fabric/compute-resources/{id}
GET /api/fabric/storage-attachments/{id}
GET /api/fabric/workspace-entries/{id}
```

- [x] Return the same durable status fields used by workspace aggregate.

- [x] Keep POST single-resource routes as accepted-operation APIs.

- [x] Add HTTP tests proving each GET returns its resource row and requires bearer auth.

Run:

```bash
npm run test:contracts
cd apps/fabric-api && go test ./internal/http ./internal/service
```

Expected: PASS.

### Task 6: Transactional Workspace Reservation

**Files:**
- Modify: `apps/fabric-api/internal/postgres/store.go`
- Modify: `apps/fabric-api/internal/service/service.go`
- Modify: `apps/fabric-api/internal/http/server_test.go`

- [x] Replace multi-step workspace reservation with a store-level transaction method.

```go
type WorkspaceReservation struct {
    Operation postgres.OperationRow
    Storage postgres.StorageVolumeRow
    Compute postgres.ComputeResourceRow
    Attachment postgres.StorageAttachmentRow
    Entry postgres.WorkspaceEntryRow
    Workspace postgres.WorkspaceRow
}

func (s *Store) CreateWorkspaceReservation(ctx context.Context, reservation WorkspaceReservation) error
```

- [x] Service `AcceptWorkspace` constructs the reservation and calls the single transactional store method.

- [x] Add a fake store test proving partial insert failure does not leave a workspace row.

Run:

```bash
cd apps/fabric-api && go test ./internal/postgres ./internal/service ./internal/http
```

Expected: PASS.

### Task 7: Documentation And Verification

**Files:**
- Modify: `README.md`
- Modify: `docs/status.md`
- Modify: `docs/architecture.md`
- Modify: this plan file

- [x] Document product mainline:

```text
Workspace create -> worker -> storage -> compute -> attach -> entry -> running workspace URL
```

- [x] Document single-resource APIs as operational/debug APIs.

- [x] Document remaining no-rollout boundary and live staging requirements.

- [x] Run full verification:

```bash
npm test
git diff --check HEAD
rg -n "os/exec|exec\\.Command|kubectl|tccli|rollout|CreateClusterNodePool\\(" apps deploy config docs README.md -g '!config/sources/**'
```

Expected:

```text
npm test exits 0
git diff --check exits 0
scan only finds documentation/prohibition/manual deployment references, not runtime shell-out
```

- [x] Commit and push to `origin/main`.
