# Status

Current status: contract-first storage-first foundation implementation.

Supported in the first implementation:

- Fabric contracts.
- Go API process.
- Catalog and readiness endpoints.
- Mutating reservation endpoints with operation receipts.
- Console-facing Workspace delivery reservation API and aggregate status API.
- No-rollout operation orchestrator and configurable background worker for accepted operations.
- Single-resource status APIs for storage, compute, attachment, and workspace entry.
- Domain safety rules.
- PostgreSQL schema.
- PostgreSQL store methods and startup migration wiring.
- Kubernetes provider boundary with fake-client tests for compute, PVC, storage attachment, workspace ingress entry, detach, and destroy.
- Tencent Cloud SDK client boundary plus live TKE NodePool create, verify, and delete provider behind an explicit mutation gate.
- Staging e2e validation gate with `blocked`, `dry_run`, and `ready_for_live` modes.
- Fake staging workspace-chain test that runs reservation -> worker -> orchestrator -> storage -> compute -> attach -> entry without touching cloud resources.
- Operator console build.

Not supported in the first implementation:

- Public GA operations.
- External payment settlement.
- Full OPL Gateway surface.
- Standalone OPL Ledger.
- GPU Workspace exposure.

Remaining risks:

- The OPL Cloud comparison is pinned to `RenDeHuang/OPL-Cloud@126e6bf8b27ef18c2d18df8d846455015e0b3ee0`; newer OPL Cloud commits need a deliberate re-baseline and contract diff.
- Readiness is still mainly configuration and environment readiness. Live PostgreSQL version checks and live Kubernetes API, storage, ingress, Tencent TKE capacity, node pool template, quota, and cluster capability checks are not complete.
- Mutating APIs now accept resource reservations and return operation receipts, including `POST /api/fabric/workspaces`. The workspace route is the product mainline; it reserves storage, compute, attachment, entry, workspace, and operation rows transactionally.
- `GET /api/fabric/workspaces/{id}` returns the reserved storage, compute, attachment, entry, and operation aggregate for Console polling. `GET` routes for each single resource expose the decomposed state for Console advanced views, operations, and recovery.
- The background worker is wired but disabled by default. When enabled, it leases accepted operations and runs the orchestrator. No real cluster worker run has been performed in this environment.
- PostgreSQL coverage includes store method compilation and startup migration wiring, but not a live database migration and constraint test lane.
- Production console hosting still needs explicit Fabric integration from OPL Cloud/Console. OPL Cloud now carries workspace gateway/proxy behavior in its Console server, while Fabric remains the storage/compute/attachment/entry API boundary.
- The Kubernetes runtime provider now covers Deployment, Service, workspace Codex Secret, PVC, attachment mount, workspace ingress entry, detach, compute destroy, and PVC destroy with fake-client tests. Reconcile, status, watch behavior, and real-cluster validation remain future work.
- The Tencent capacity provider now creates, verifies, and deletes TKE NodePools through the Tencent Cloud Go SDK when `OPL_TKE_ALLOW_NODEPOOL_MUTATION=true`. It is still gated by default and has not been exercised against the live staging cluster from this environment.
- The deployment manifest has minimal RBAC for the current provider only; future read, watch, patch, and update flows must expand it deliberately.
- OPL Cloud catalog sections for environment templates, connectors, and agent packages are not yet implemented in OPL Fabric.
- OPL Cloud deployment contract fields for ingress, image pull secrets, TLS, Tencent registry, Codex runtime config, TKE node pool launch config, autoscaling config, and production diagnostics are not fully represented by the current deployment skeleton.
- No real-cluster validation has run in this environment; current Kubernetes checks are fake-client or YAML structural checks.
- No rollout was performed for Phase 3-7. The default path is controlled dry-run. Live staging e2e remains blocked until live PostgreSQL, kubeconfig or in-cluster config, TKE, TCR, storage class, ingress class, Workspace image, worker enablement, and explicit live mutation flags are verified.
- Central config records medopl-3 TKE and Codex workspace inputs plus the latest OPL Cloud Tencent node pool knobs. The Kubernetes provider covers the create/destroy primitives for the main chain with fake-client tests; reconcile, watch, status, and live readiness checks remain incomplete.
- `OPL_CODEX_API_KEY` is optional until workspace Codex bootstrap is enabled for a published mutating compute API.
- The latest OPL Cloud commit reverts its TKE NodePool goal work, so Fabric should continue with the Go client-go plus Tencent Cloud Go SDK split instead of importing OPL Cloud's current JavaScript provider/runtime approach.
