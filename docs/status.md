# Status

Current status: contract-first storage-first foundation implementation.

Supported in the first implementation:

- Fabric contracts.
- Go API process.
- Catalog and readiness endpoints.
- Mutating reservation endpoints with operation receipts.
- Console-facing Workspace delivery reservation API and aggregate status API.
- No-rollout operation orchestrator for accepted operations.
- Domain safety rules.
- PostgreSQL schema.
- PostgreSQL store methods and startup migration wiring.
- Kubernetes provider boundary with fake-client tests for compute, PVC, storage attachment, workspace ingress entry, detach, and destroy.
- Tencent Cloud SDK client boundary plus NodePool dry-plan resolver for TKE capacity operations.
- Staging e2e validation gate that blocks live mutation unless explicit flags and required cloud inputs are present.
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
- Mutating APIs now accept resource reservations and return operation receipts, including `POST /api/fabric/workspaces`, but they do not yet run the asynchronous orchestrator or mutate Kubernetes/Tencent Cloud resources.
- `GET /api/fabric/workspaces/{id}` returns the reserved storage, compute, attachment, entry, and operation aggregate for Console polling.
- The orchestrator is implemented as a control-plane component, but it is not yet wired as a background worker in the API process.
- PostgreSQL coverage includes store method compilation and startup migration wiring, but not a live database migration and constraint test lane.
- Production console hosting still needs explicit Fabric integration from OPL Cloud/Console. OPL Cloud now carries workspace gateway/proxy behavior in its Console server, while Fabric remains the storage/compute/attachment/entry API boundary.
- The Kubernetes runtime provider now covers Deployment, Service, workspace Codex Secret, PVC, attachment mount, workspace ingress entry, detach, compute destroy, and PVC destroy with fake-client tests. Reconcile, status, watch behavior, and real-cluster validation remain future work.
- The Tencent capacity provider boundary currently constructs a TKE SDK client and validates NodePool dry plans, but does not yet create, scale, or verify node pools.
- The deployment manifest has minimal RBAC for the current provider only; future read, watch, patch, and update flows must expand it deliberately.
- OPL Cloud catalog sections for environment templates, connectors, and agent packages are not yet implemented in OPL Fabric.
- OPL Cloud deployment contract fields for ingress, image pull secrets, TLS, Tencent registry, Codex runtime config, TKE node pool launch config, autoscaling config, and production diagnostics are not fully represented by the current deployment skeleton.
- No real-cluster validation has run in this environment; current Kubernetes checks are fake-client or YAML structural checks.
- No rollout was performed for Phase 3-7. Staging e2e remains blocked until live PostgreSQL, kubeconfig, TKE, TCR, storage class, ingress class, and explicit live mutation flags are verified.
- Central config now records medopl-3 TKE and Codex workspace inputs plus the latest OPL Cloud Tencent node pool knobs, but provider behavior still needs full PVC, Secret, Ingress, attachment, Workspace entry, reconcile, and status implementation through Go client-go.
- `OPL_CODEX_API_KEY` is optional until workspace Codex bootstrap is enabled for a published mutating compute API.
- The latest OPL Cloud commit reverts its TKE NodePool goal work, so Fabric should continue with the Go client-go plus Tencent Cloud Go SDK split instead of importing OPL Cloud's current JavaScript provider/runtime approach.
