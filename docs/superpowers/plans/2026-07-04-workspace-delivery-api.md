# Workspace Delivery API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provide the first Console-facing OPL Fabric Workspace API so OPL Console can request workspace delivery through Fabric instead of owning cloud execution details.

**Architecture:** `POST /api/fabric/workspaces` creates a durable workspace aggregate reservation and returns an operation receipt. Fabric records storage, compute, attachment, and entry rows up front, then later worker/orchestrator execution can fulfill them. This phase is no-rollout: it does not call live Kubernetes or Tencent Cloud APIs.

**Tech Stack:** Go API, PostgreSQL store interfaces, OpenAPI + JSON Schema contracts, existing operation receipt pattern.

---

### Task 1: Contract

**Files:**
- Modify: `contracts/fabric-api.openapi.json`
- Create: `contracts/fabric-workspace.schema.json`
- Modify: `tests/contracts/contracts.test.mjs`

- [x] Add Workspace schema with storage, compute, attachment, entry, and operation fields.
- [x] Add `POST /api/fabric/workspaces`.
- [x] Add `GET /api/fabric/workspaces/{id}`.
- [x] Keep all mutating routes requiring `Idempotency-Key` and `X-Correlation-Id`.

### Task 2: Store Boundary

**Files:**
- Modify: `apps/fabric-api/internal/postgres/schema.sql`
- Modify: `apps/fabric-api/internal/postgres/store.go`
- Modify: `apps/fabric-api/internal/postgres/store_test.go`

- [x] Add `workspaces` table with links to compute, storage, attachment, entry, and operation.
- [x] Add `CreateWorkspace` and `GetWorkspace` store methods.
- [x] Add nil-store tests for the new methods.

### Task 3: Service and HTTP

**Files:**
- Modify: `apps/fabric-api/internal/service/service.go`
- Modify: `apps/fabric-api/internal/http/server.go`
- Modify: `apps/fabric-api/internal/http/server_test.go`

- [x] Add `AcceptWorkspace` service method that creates storage, compute, attachment, entry, workspace, and operation rows.
- [x] Add `Workspace` status service method.
- [x] Add HTTP handlers for `POST /api/fabric/workspaces` and `GET /api/fabric/workspaces/{id}`.

### Task 4: Verification

**Commands:**
- `npm test`
- `git diff --check HEAD`
- Retired narrative scan excluding `config/sources/**`
- Namespace scan excluding `config/sources/**`

- [x] All tests pass.
- [x] No rollout or live cloud mutation is introduced.
- [x] Commit and push to `origin/main`.
