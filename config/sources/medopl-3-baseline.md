# medopl-3 Baseline

Source path: `/home/dev/medopl-3`

Observed commit:

- `d2c7474 Bootstrap Codex config for TKE workspaces`
- Parent feature commit: `25403cc Inject Codex provider settings into TKE workspaces`

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
- TCR image references must match the configured registry.
- Workspace runtime exposes port `3000`.
- Workspace persistent data uses `/data`.
- Workspace projects use `/projects`.
- Codex config is bootstrapped under `/data/codex`.
- Sensitive values must use secret refs or external secret files.
- VolumeSnapshot support depends on `snapshot.storage.k8s.io`.

Not imported:

- Commercial Console users and billing runtime keys.
- Real secrets from `.runtime`, `.env`, or `/home/dev/.secrets`.
- JavaScript `kubectl` execution as a runtime implementation pattern.
