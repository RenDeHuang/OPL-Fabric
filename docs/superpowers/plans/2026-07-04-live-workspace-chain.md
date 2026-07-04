# Live Workspace Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect the real workspace business chain so Fabric can create and verify Tencent TKE NodePools for dedicated compute, then run storage -> compute -> attach -> entry through the worker under explicit live gates.

**Architecture:** Keep OPL Console on `POST /api/fabric/workspaces`. The worker remains the only runtime executor. Tencent Cloud capacity is a separate Go SDK provider called before Kubernetes compute creation when `isolationMode=dedicated_nodepool`; shared-pool compute skips NodePool creation. Live mutation stays gated by config and staging/production readiness inputs.

**Tech Stack:** Go, PostgreSQL, Tencent Cloud Go SDK, Kubernetes client-go, OpenAPI/JSON Schema, existing Fabric worker/orchestrator/runtime boundaries.

---

### Task 1: Tencent Live NodePool Provider

**Files:**
- Modify: `apps/fabric-api/internal/tencentcloud/tke.go`
- Modify: `apps/fabric-api/internal/tencentcloud/tke_test.go`

- [x] Add a `TKEAPI` interface with `CreateClusterNodePoolWithContext`, `DescribeClusterNodePoolsWithContext`, and `DeleteClusterNodePoolWithContext`.
- [x] Add `NodePoolProvider` with `EnsureNodePool`, `VerifyNodePool`, and `DeleteNodePool`.
- [x] Build `CreateClusterNodePoolRequest` from the existing launch/autoscaling JSON strings.
- [x] Refuse mutation with `ErrNodePoolMutationNotAllowed` when `MutationAllowed=false`.
- [x] Add tests using a fake TKE client proving create/verify/delete request shape and gate behavior.

Run:

```bash
cd apps/fabric-api && go test ./internal/tencentcloud
```

Expected: PASS.

### Task 2: Capacity Runtime Boundary

**Files:**
- Modify: `apps/fabric-api/internal/fabricruntime/kubernetes.go`
- Modify: `apps/fabric-api/internal/fabricruntime/kubernetes_test.go`
- Modify: `apps/fabric-api/internal/orchestrator/orchestrator.go`
- Modify: `apps/fabric-api/internal/orchestrator/orchestrator_test.go`

- [x] Add a `CapacityProvider` interface to `fabricruntime`.
- [x] In `CreateCompute`, call capacity only for `IsolationMode == "dedicated_nodepool"` or `CapacityPoolID == "dedicated-nodepool-template"`.
- [x] Store the returned NodePool ID in `RuntimeComputeResult.NodePoolID` and then in `compute_resources.node_pool_id`.
- [x] In `DestroyCompute`, delete the dedicated NodePool only when the compute row has a `NodePoolID`.
- [x] Add fake tests proving shared pool skips capacity calls and dedicated compute calls create/verify/delete.

Run:

```bash
cd apps/fabric-api && go test ./internal/fabricruntime ./internal/orchestrator
```

Expected: PASS.

### Task 3: Wire Live Provider Into API Worker

**Files:**
- Modify: `apps/fabric-api/cmd/fabric-api/main.go`
- Modify: `apps/fabric-api/internal/config/config.go`
- Modify: `apps/fabric-api/internal/config/config_test.go`
- Modify: `config/fabric.env.example`
- Modify: `deploy/k8s/opl-fabric-api.yaml`

- [x] Build a Tencent SDK client in `startWorker` only when worker is enabled and Tencent inputs are present.
- [x] Pass `NodePoolProvider` into `fabricruntime.KubernetesRuntime`.
- [x] Keep `OPL_TKE_ALLOW_NODEPOOL_MUTATION=false` as the safe example default, while allowing explicit true for staging/prod.
- [x] Add config tests proving mutation gate and NodePool JSON inputs are loaded.

Run:

```bash
cd apps/fabric-api && go test ./cmd/fabric-api ./internal/config
```

Expected: PASS.

### Task 4: Staging E2E Gate

**Files:**
- Modify: `apps/fabric-api/internal/staging/gate.go`
- Modify: `apps/fabric-api/internal/staging/gate_test.go`
- Create: `apps/fabric-api/internal/staging/workspace_chain_test.go`

- [x] Add a staging gate result that distinguishes `dry_run`, `ready_for_live`, and `blocked`.
- [x] Require PostgreSQL, kubeconfig/in-cluster config, TKE, TCR, storage class, ingress class, workspace image, and explicit live flags before live e2e.
- [x] Add a fake e2e test that runs workspace reservation -> worker -> orchestrator -> storage -> compute -> attach -> entry without real cloud.
- [x] Add a skipped live e2e skeleton that only runs when `OPL_STAGING_E2E_ALLOW_LIVE=true` and `OPL_FABRIC_WORKER_ENABLED=true`.

Run:

```bash
cd apps/fabric-api && go test ./internal/staging
```

Expected: PASS, with live e2e skipped unless live flags are set.

### Task 5: Documentation, Verification, Push

**Files:**
- Modify: `README.md`
- Modify: `docs/status.md`
- Modify: `docs/architecture.md`
- Modify: this plan file

- [x] Document controlled live behavior: who created resources, cluster ID, NodePool ID, operation refs, provider refs, and delete confirmation.
- [x] Document that Console creates through Fabric routes and Ledger records Fabric refs.
- [x] Run full verification:

```bash
npm test
git diff --check HEAD
rg -n "os/exec|exec\\.Command|kubectl|tccli|rollout" apps deploy config docs README.md -g '!config/sources/**'
```

Expected:

```text
npm test exits 0
git diff --check exits 0
scan only finds docs/manual commands, not runtime shell-out
```

- [x] Commit and push to `origin/main`.
