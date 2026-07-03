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

Remaining risks:

- The OPL Cloud comparison is pinned to `RenDeHuang/OPL-Cloud@2985bfdaa592a0644da5fdb0c11a877785a85155`; newer OPL Cloud commits need a deliberate re-baseline and contract diff.
- Readiness is still mainly configuration and environment readiness. Live PostgreSQL migration/version checks and live Kubernetes API, storage, ingress, snapshot, and cluster capability checks are not complete.
- Mutating APIs are intentionally not published yet. Compute, storage, attachment, route, backup, and restore writes still need contract-first request and response schemas, idempotency, authority, operation receipts, and tests before handlers are exposed.
- PostgreSQL coverage is mostly schema/static verification, not a live database migration and constraint test lane.
- Runtime startup does not yet open PostgreSQL or run migrations, even though the store migration code exists.
- Production console hosting still needs server-side auth, proxy, and session design. The Vite proxy is a local development path only.
- The Kubernetes provider currently covers Deployment and Service creation/deletion. PVC, Ingress, Secret, VolumeSnapshot, route, backup, restore, reconcile, status, and watch behavior remain future work.
- The deployment manifest has minimal RBAC for the current provider only; future read, watch, patch, and update flows must expand it deliberately.
- OPL Cloud catalog sections for environment templates, connectors, and agent packages are not yet implemented in OPL Fabric.
- OPL Cloud deployment contract fields for ingress, image pull secrets, snapshot classes, TLS, Tencent registry, and production diagnostics are not fully represented by the current deployment skeleton.
- No real-cluster validation has run in this environment; current Kubernetes checks are fake-client or YAML structural checks.
- Central config now records medopl-3 TKE and Codex workspace inputs, but provider behavior still needs full PVC, Secret, Ingress, VolumeSnapshot, backup, restore, reconcile, and status implementation through Go client-go.
- `OPL_CODEX_API_KEY` is optional until workspace Codex bootstrap is enabled for a published mutating compute API.
