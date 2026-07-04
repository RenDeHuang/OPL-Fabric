# medopl-3 and OPL Cloud Baseline

Source path: `/home/dev/medopl-3`

Observed commit:

- `d2c7474 Bootstrap Codex config for TKE workspaces`
- Parent feature commit: `25403cc Inject Codex provider settings into TKE workspaces`

Active OPL Cloud comparison:

- Repository: `RenDeHuang/OPL-Cloud`
- Commit: `126e6bf8b27ef18c2d18df8d846455015e0b3ee0`
- Commit message: `revert: stop tke node pool goal work`

Files read:

- `deploy/tke/opl-cloud-production.env.example`
- `deploy/production-manifest.example.json`
- `packages/console/src/production-manifest.js`
- `packages/console/src/production-readiness.js`
- `packages/fabric/src/runtime-providers/tencent-tke.js`
- `docs/runtime/tke-production-deployment.md`
- `docs/runtime/production-runbook.md`

Imported concepts:

- Tencent TKE is the production provider.
- OPL Fabric separates Kubernetes runtime operations from Tencent Cloud capacity operations.
- Storage is persistent; compute is rebuildable; Workspace entry comes after storage attachment.
- TCR image references must match the configured registry.
- Workspace runtime exposes port `3000`.
- Workspace persistent data uses `/data`.
- Workspace projects use `/projects`.
- Codex config is bootstrapped under `/data/codex`.
- Sensitive values must use secret refs or external secret files.
- Tencent TKE NodePool creation needs Tencent region, cluster ID, mutation credentials, instance charge type, autoscaling group parameters, and launch configuration parameters.

Not imported:

- Commercial Console users and billing runtime keys.
- Real secrets from `.runtime`, `.env`, or `/home/dev/.secrets`.
- JavaScript `kubectl` execution as a runtime implementation pattern.
- `tccli` execution as a runtime implementation pattern.
- The removed OPL Cloud copy-based storage lifecycle contracts.
