# OPL Cloud 126e6bf Production API Baseline

Source path: `/home/dev/opl-cloud`

Observed commit:

- `126e6bf8b27ef18c2d18df8d846455015e0b3ee0`
- Commit date: `2026-07-04T12:45:42+08:00`
- Commit message: `revert: stop tke node pool goal work`

Files copied into this source snapshot:

- `config/sources/opl-cloud-126e6bf/opl-cloud-route-api-contract.json`
- `config/sources/opl-cloud-126e6bf/opl-cloud-deployment-contract.json`
- `config/sources/opl-cloud-126e6bf/opl-cloud-product-contract.json`
- `config/sources/opl-cloud-126e6bf/opl-cloud-resource-catalog.js`

Imported concepts:

- Lab Owner APIs model compute resources, storage volumes, storage attachments, and Workspace entries as separate business objects.
- Workspace entry creation requires an attached storage/compute pair.
- Storage remains retained across compute destruction and rebuild.
- Workspace URLs use a gateway path with a trailing slash: `/w/<workspaceId>/`.
- Console may call Fabric only through package boundaries or published service APIs.
- Active resource APIs replace retired combined lifecycle routes.

Not imported:

- JavaScript provider runtime.
- `kubectl` shell-out as normal runtime.
- `tccli` shell-out as normal runtime.
- The reverted OPL Cloud TKE NodePool goal work as a long-term Fabric implementation.
- Feature-gated copy-based storage lifecycle routes.

Current OPL Fabric divergence:

- Default staging namespace is `opl-fabric`, not the source `opl-cloud` namespace.
- Mutating API contracts return accepted operation receipts. Real orchestration starts in a later phase.
- Tencent NodePool mutation is allowed for staging through `OPL_TKE_ALLOW_NODEPOOL_MUTATION=true`, but only the future Tencent Cloud Go SDK resolver may use it.
- OPL Cloud currently carries gateway/proxy behavior in its Console server; OPL Fabric keeps the runtime provider and API contract independent so OPL Cloud/Console can call it explicitly.
