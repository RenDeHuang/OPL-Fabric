# Status

Current status: contract-first storage-first foundation implementation.

Supported in the first implementation:

- Fabric contracts.
- Go API process.
- Catalog and readiness endpoints.
- Domain safety rules.
- PostgreSQL schema.
- Kubernetes provider boundary with fake-client tests.
- Tencent Cloud SDK client boundary for TKE capacity operations.
- Operator console build.

Not supported in the first implementation:

- Public GA operations.
- External payment settlement.
- Full OPL Gateway surface.
- Standalone OPL Ledger.
- GPU Workspace exposure.

Remaining risks:

- The OPL Cloud comparison is pinned to `RenDeHuang/OPL-Cloud@854b047a28148f84912924856975b8c1f0077448`; newer OPL Cloud commits need a deliberate re-baseline and contract diff.
- Readiness is still mainly configuration and environment readiness. Live PostgreSQL migration/version checks and live Kubernetes API, storage, ingress, Tencent TKE capacity, node pool template, quota, and cluster capability checks are not complete.
- Mutating APIs are intentionally not published yet. Compute create/destroy, storage create/destroy, attach/detach, Workspace entry, and operation status writes still need contract-first request and response schemas, idempotency, authority, operation receipts, and tests before handlers are exposed.
- PostgreSQL coverage is mostly schema/static verification, not a live database migration and constraint test lane.
- Runtime startup does not yet open PostgreSQL or run migrations, even though the store migration code exists.
- Production console hosting still needs server-side auth, proxy, and session design. The Vite proxy is a local development path only.
- The Kubernetes runtime provider currently covers Deployment, Service, and workspace Codex Secret creation. PVC, Ingress, attachment mount, Workspace entry routing, reconcile, status, and watch behavior remain future work.
- The Tencent capacity provider boundary currently constructs a TKE SDK client but does not yet create, scale, or verify node pools.
- The deployment manifest has minimal RBAC for the current provider only; future read, watch, patch, and update flows must expand it deliberately.
- OPL Cloud catalog sections for environment templates, connectors, and agent packages are not yet implemented in OPL Fabric.
- OPL Cloud deployment contract fields for ingress, image pull secrets, TLS, Tencent registry, TKE node pool launch config, autoscaling config, and production diagnostics are not fully represented by the current deployment skeleton.
- No real-cluster validation has run in this environment; current Kubernetes checks are fake-client or YAML structural checks.
- Central config now records medopl-3 TKE and Codex workspace inputs plus the latest OPL Cloud Tencent node pool knobs, but provider behavior still needs full PVC, Secret, Ingress, attachment, Workspace entry, reconcile, and status implementation through Go client-go.
- `OPL_CODEX_API_KEY` is optional until workspace Codex bootstrap is enabled for a published mutating compute API.
