# Production API Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Phase 0-2 for the OPL Fabric production API: source snapshots, mutating API contracts, PostgreSQL store methods, and startup migration wiring.

**Architecture:** OPL Fabric remains contract-first. Mutating HTTP routes accept requests, require idempotency and correlation headers, persist resource reservations plus accepted operations in PostgreSQL, and return operation receipts. Actual Kubernetes runtime orchestration and Tencent NodePool mutation are intentionally left to later phases.

**Tech Stack:** React + TypeScript console, Go API, PostgreSQL, Kubernetes Go client-go, Tencent Cloud Go SDK, OpenAPI + JSON Schema, `config/` plus `OPL_FABRIC_CONFIG_DIR`.

---

### Task 1: Source Baseline And Narrative Cleanup

**Files:**
- Create: `config/sources/opl-cloud-126e6bf-production-api-baseline.md`
- Copy source examples into: `config/sources/medopl-3/` and `config/sources/opl-cloud-126e6bf/`
- Modify: `docs/decisions.md`
- Modify: `docs/status.md`
- Modify: `config/sources/medopl-3-baseline.md`
- Delete: superseded `docs/superpowers/specs/2026-07-03-opl-fabric-split-design.md`
- Delete: superseded `docs/superpowers/plans/2026-07-03-opl-fabric-contract-first-foundation.md`
- Delete: superseded `docs/superpowers/plans/2026-07-04-centralized-fabric-config.md`

- [x] Record active OPL Cloud baseline as `RenDeHuang/OPL-Cloud@126e6bf8b27ef18c2d18df8d846455015e0b3ee0`.
- [x] Preserve source-derived config examples without importing secrets.
- [x] Keep retired runtime and lifecycle wording out of active narrative.
- [x] Set default Kubernetes namespace to `opl-fabric`.
- [x] Sync the latest OPL Cloud workspace gateway path pattern `/w/<workspaceId>/`.

### Task 2: Production API Contract

**Files:**
- Modify: `contracts/fabric-api.openapi.json`
- Create: `contracts/fabric-operation-receipt.schema.json`
- Test: `tests/contracts/contracts.test.mjs`

- [x] Add operation receipt schema with `operationId`, `state`, `resourceKind`, and `resourceId`.
- [x] Add required `Idempotency-Key` and `X-Correlation-Id` header parameters for every mutating operation.
- [x] Add mutating contracts for storage volumes, compute resources, storage attachments, workspace entries, compute destroy, storage destroy, attachment detach, and operation lookup.
- [x] Keep published OpenAPI routes aligned with implemented HTTP handlers.

### Task 3: PostgreSQL Store And Startup Migration

**Files:**
- Modify: `apps/fabric-api/internal/postgres/schema.sql`
- Modify: `apps/fabric-api/internal/postgres/store.go`
- Modify: `apps/fabric-api/internal/postgres/store_test.go`
- Modify: `apps/fabric-api/cmd/fabric-api/main.go`

- [x] Add owner fields to attachment and workspace entry persistence.
- [x] Add store insert/get methods for accepted operations and resource reservations.
- [x] Wire API startup to open PostgreSQL and run `Store.Migrate` when `DATABASE_URL` is configured.

### Task 4: Accepted Operation HTTP Handlers

**Files:**
- Modify: `apps/fabric-api/internal/domain/types.go`
- Modify: `apps/fabric-api/internal/service/service.go`
- Modify: `apps/fabric-api/internal/service/service_test.go`
- Modify: `apps/fabric-api/internal/http/server.go`
- Modify: `apps/fabric-api/internal/http/server_test.go`

- [x] Add request/receipt types for resource reservation operations.
- [x] Add service methods that persist resource rows and accepted operation rows.
- [x] Add HTTP handlers that validate headers, decode JSON, call service methods, and return `202 Accepted`.
- [x] Keep later orchestration out of scope; accepted operations do not mutate Kubernetes or Tencent Cloud yet.

### Task 5: Verification

**Commands:**
- `npm test`
- `git diff --check HEAD`
- Run a retired-term scan over active docs, contracts, config, deployment files, tests, and app code while excluding copied source snapshots.
- `git status --short --branch`

- [x] All tests pass.
- [x] Whitespace check passes.
- [x] Old active narrative and schema names do not reappear.
- [ ] Commit and push to `origin/main`.
