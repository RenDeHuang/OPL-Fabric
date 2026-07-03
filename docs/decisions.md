# Decisions

## 2026-07-03: OPL Fabric split

OPL Fabric is split as an independent Fabric service, not a full OPL Cloud rewrite.

The implementation is contract-first. The backend is Go, persistence is PostgreSQL, Kubernetes operations use client-go, and the frontend is React TypeScript.

one-person-lab is used as the development framework reference for contracts, lifecycle ledgers, evidence, human gates, readiness, and ownership boundaries.

## 2026-07-03: OPL Cloud baseline pinned

The OPL Cloud baseline for this split is:

- Repository: `RenDeHuang/OPL-Cloud`
- Branch: `main`
- Commit: `2985bfdaa592a0644da5fdb0c11a877785a85155`
- Commit date: `2026-07-03T14:33:40Z`
- Commit message: `fix: tolerate workspace websocket resets`
- Commit URL: `https://github.com/RenDeHuang/OPL-Cloud/commit/2985bfdaa592a0644da5fdb0c11a877785a85155`

The source files used as the split reference are:

- `packages/contracts/opl-cloud-product-contract.json`
- `packages/contracts/opl-cloud-fabric-resource-catalog-contract.json`
- `packages/contracts/opl-cloud-workspace-lifecycle-contract.json`
- `packages/contracts/opl-cloud-storage-backup-contract.json`
- `packages/contracts/opl-cloud-deployment-contract.json`
- `packages/fabric/src/index.js`
- `packages/fabric/src/resource-catalog.js`
- `packages/fabric/src/runtime-provider-factory.js`
- `packages/fabric/src/runtime-providers/local-docker.js`
- `packages/fabric/src/runtime-providers/tencent-tke.js`

Future OPL Cloud changes do not automatically redefine OPL Fabric contracts. Re-baseline by recording the new OPL Cloud commit, diffing the files above plus any newly relevant contracts, and then making explicit contract-first changes in OPL Fabric.
