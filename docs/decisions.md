# Decisions

## 2026-07-03: OPL Fabric split

OPL Fabric is split as an independent Fabric service, not a full OPL Cloud rewrite.

The implementation is contract-first. The backend is Go, persistence is PostgreSQL, Kubernetes operations use client-go, and the frontend is React TypeScript.

one-person-lab is used as the development framework reference for contracts, lifecycle ledgers, evidence, human gates, readiness, and ownership boundaries.

## 2026-07-03: OPL Cloud baseline pinned

The original OPL Cloud baseline for this split was:

- Repository: `RenDeHuang/OPL-Cloud`
- Branch: `main`
- Commit: `2985bfdaa592a0644da5fdb0c11a877785a85155`
- Commit date: `2026-07-03T14:33:40Z`
- Commit message: `fix: tolerate workspace websocket resets`
- Commit URL: `https://github.com/RenDeHuang/OPL-Cloud/commit/2985bfdaa592a0644da5fdb0c11a877785a85155`

That baseline is superseded by the active storage-first re-baseline below. It remains here only to explain the original repository split.

Future OPL Cloud changes do not automatically redefine OPL Fabric contracts. Re-baseline by recording the new OPL Cloud commit, diffing newly relevant contracts and provider files, and then making explicit contract-first changes in OPL Fabric.

## 2026-07-04: OPL Cloud storage-first re-baseline

The active OPL Cloud comparison is:

- Repository: `RenDeHuang/OPL-Cloud`
- Branch: `main`
- Commit: `854b047a28148f84912924856975b8c1f0077448`
- Commit message: `fix: avoid custom pod cidr node pool setting`
- Commit URL: `https://github.com/RenDeHuang/OPL-Cloud/commit/854b047a28148f84912924856975b8c1f0077448`

This baseline changes the OPL Fabric direction:

- Storage is the durable resource.
- Compute is rebuildable capacity.
- Storage attachment precedes Workspace entry creation.
- TKE NodePool capacity is a cloud capacity concern, not the Kubernetes runtime object itself.
- The old workspace bundle and copy-based storage lifecycle narrative is not carried forward as a compatibility layer.

The stable implementation stack is fixed as React + TypeScript frontend, Go backend, PostgreSQL durable store, Kubernetes Go client-go runtime provider, Tencent Cloud Go SDK capacity provider, OpenAPI + JSON Schema contracts, and `config/` with `OPL_FABRIC_CONFIG_DIR` for runtime configuration.

Normal runtime must not depend on `kubectl` shell-out, `tccli` shell-out, or JavaScript provider runtime.

## 2026-07-04: Central Fabric config directory

OPL Fabric uses `config/` as the default configuration root. Operators can move the directory and set `OPL_FABRIC_CONFIG_DIR` to the new path.

The initial config catalog imports public deployment and provider key names from `/home/dev/medopl-3` at commit `d2c7474deb6deb39daf81232f563a5f39c0fdd16`. Real secrets are not imported; only key names, defaults, workspace runtime paths, readiness checks, and secretRef shapes are retained.
