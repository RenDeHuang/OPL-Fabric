# Phases 3-7 Control Plane Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the non-rollout control-plane implementation for Phase 3-7: operation orchestration, Kubernetes fake-client runtime actions, Tencent NodePool dry planning, and staging e2e gates.

**Architecture:** The API remains contract-first. Accepted operations are persisted first, then a separate orchestrator applies resource actions through narrow provider interfaces. This pass must not connect to staging, must not call live Tencent mutation APIs, and must not roll out Kubernetes manifests.

**Tech Stack:** Go, PostgreSQL store interfaces, Kubernetes Go client-go fake tests, Tencent Cloud Go SDK boundary, OpenAPI + JSON Schema, `config/` runtime keys.

---

### Task 1: Operation Orchestrator Boundary

**Files:**
- Create: `apps/fabric-api/internal/orchestrator/orchestrator.go`
- Create: `apps/fabric-api/internal/orchestrator/orchestrator_test.go`
- Modify: `apps/fabric-api/internal/postgres/store.go`
- Modify: `apps/fabric-api/internal/postgres/store_test.go`

- [x] Add failing tests for applying accepted storage, compute, attachment, workspace entry, and destroy operations through fake providers.
- [x] Add store row lookup/update interfaces needed by the orchestrator.
- [x] Implement an orchestrator that updates operation state from `accepted` to `applying`, then `succeeded` or `failed`.
- [x] Keep operation execution out of HTTP request handlers.

### Task 2: Kubernetes Runtime Provider

**Files:**
- Modify: `apps/fabric-api/internal/k8s/provider.go`
- Modify: `apps/fabric-api/internal/k8s/provider_test.go`

- [x] Add failing fake-client tests for PVC creation.
- [x] Add failing fake-client tests for storage attachment patching Deployment volume mounts.
- [x] Add failing fake-client tests for workspace gateway ingress entry.
- [x] Add failing fake-client tests for detach, compute destroy, and storage destroy.
- [x] Implement only client-go runtime behavior; do not shell out to `kubectl`.

### Task 3: Tencent NodePool Resolver Dry Plan

**Files:**
- Modify: `apps/fabric-api/internal/tencentcloud/tke.go`
- Modify: `apps/fabric-api/internal/tencentcloud/tke_test.go`

- [x] Add tests for validating cluster id, region, credentials, launch JSON, autoscaling JSON, and mutation gate.
- [x] Implement a dry-plan resolver that returns the Tencent SDK request boundary without calling live APIs.
- [x] Keep real create/delete/scale NodePool mutation behind a later explicit staging gate.

### Task 4: Staging E2E Gate

**Files:**
- Create: `apps/fabric-api/internal/staging/gate.go`
- Create: `apps/fabric-api/internal/staging/gate_test.go`
- Modify: `docs/status.md`

- [x] Add tests showing staging e2e is blocked unless PostgreSQL, kubeconfig, TKE, TCR, storage class, ingress class, and explicit live mutation flags are present.
- [x] Implement the gate as validation-only code.
- [x] Document that no rollout was performed in this phase.

### Task 5: Verification And Commit

**Commands:**
- `npm test`
- `git diff --check HEAD`
- Retired narrative scan excluding `config/sources/**`
- Namespace scan excluding `config/sources/**`
- `git status --short --branch`

- [x] All tests pass.
- [x] Whitespace check passes.
- [x] No rollout, staging mutation, `kubectl`, `tccli`, or JS runtime dependency is introduced into normal runtime code.
- [ ] Commit and push to `origin/main`.
