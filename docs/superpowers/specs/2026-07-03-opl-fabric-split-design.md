# OPL Fabric Split Design

Date: 2026-07-03

Status: approved for implementation planning

## Goal

Split `OPL Fabric` out as an independent contract-first cloud resource service.

The service owns the Fabric control plane for OPL Cloud: resource catalog, runtime readiness, compute lifecycle, storage lifecycle, storage attachment, Workspace routing, storage backup, storage restore, Kubernetes provider execution, and provider evidence.

The service does not own OPL Console commercial flows, wallet and billing truth, OPL Ledger reconciliation, OPL Gateway AI routing, `one-person-lab` framework internals, or `one-person-lab-app` WebUI behavior.

## Source Inputs

This design is based on:

- `RenDeHuang/OPL-Cloud@2985bfdaa592a0644da5fdb0c11a877785a85155` contracts and implementation shape from branch `main`.
- OPL Cloud commit date `2026-07-03T14:33:40Z`.
- OPL Cloud commit message `fix: tolerate workspace websocket resets`.
- `OPL-Cloud/packages/contracts/opl-cloud-product-contract.json`.
- `OPL-Cloud/packages/contracts/opl-cloud-fabric-resource-catalog-contract.json`.
- `OPL-Cloud/packages/contracts/opl-cloud-workspace-lifecycle-contract.json`.
- `OPL-Cloud/packages/contracts/opl-cloud-storage-backup-contract.json`.
- `OPL-Cloud/packages/contracts/opl-cloud-deployment-contract.json`.
- `OPL-Cloud/packages/fabric/src/**`, especially the Local Docker and Tencent TKE providers.
- `gaofeng21cn/one-person-lab` as the development framework reference for contract-light truth, lifecycle ledger, evidence, human gates, readiness, blocker reporting, and ownership boundaries.

`OPL-Fabric` starts as an empty repository. This design treats `OPL-Cloud` as the source of current product contracts and old implementation behavior, not as a codebase to copy directly.

Future `OPL-Cloud` updates require an explicit re-baseline and contract diff before changing `OPL-Fabric` behavior.

## Fixed Technology Stack

- Frontend: React + TypeScript.
- Backend: Go.
- DB: PostgreSQL.
- Kubernetes: Go `client-go`.

Node, Vite, and JavaScript remain relevant only as legacy implementation references from `OPL-Cloud`. The new backend must not shell out to `kubectl` for normal provider actions.

## Product Boundary

### Owned By OPL Fabric

- Fabric resource catalog.
- Runtime provider readiness.
- ComputeResource lifecycle.
- StorageVolume lifecycle.
- StorageAttachment lifecycle.
- WorkspaceRoute lifecycle.
- Kubernetes Deployment, Secret, PVC, Service, Ingress, Endpoint, and VolumeSnapshot mechanics.
- Storage backup, restore-to-new-storage mechanics, and retention evidence.
- Provider operation attempts, receipts, events, and verification evidence.
- Operator-facing Fabric console for readiness, resource status, operation history, and provider blockers.

### Not Owned By OPL Fabric

- OPL Console login, session, PI account, organization, member, support, and commercial UI.
- Wallet balances, holds, hourly charging, billing ledger, manual top-up, and Tencent bill reconciliation.
- OPL Gateway provider routing, AI usage policy, token policy, and request metering.
- `one-person-lab` runtime, activation layer, domain modules, sessions, progress, and artifacts.
- `one-person-lab-app` WebUI runtime behavior inside a Workspace container.
- External payment settlement.
- GPU Workspace product exposure before a verified GPU node pool exists.

Console may call Fabric through published HTTP APIs or generated clients. Fabric must not import Console or Ledger implementation code.

## one-person-lab Framework Adoption

`one-person-lab` is used as the development framework model, not as a runtime dependency to embed.

Fabric adopts these framework rules:

- Machine-readable truth lives in `contracts/`.
- Narrative truth lives in `docs/`.
- Current docs describe current truth only; history cannot redefine the active boundary.
- Each long-running provider action has an operation attempt, event envelope, lifecycle ledger entry, verification result, and evidence refs.
- Readiness responses include exact blockers and repair hints, not just a boolean.
- High-risk actions have an explicit human gate or an idempotency key with recorded authority.
- Projection surfaces are read-only when they report truth owned by another system.
- Fabric owns its durable truth only for Fabric resources and provider execution evidence.

The framework concepts map into Fabric as follows:

| one-person-lab concept | Fabric equivalent |
| --- | --- |
| family event envelope | `fabric-event-envelope` for operation correlation |
| lifecycle ledger | `fabric-lifecycle-ledger` for dry-run, apply, verify, and receipts |
| human gate | `fabric-human-gate` for destructive or shared-route actions |
| runtime supervision | `fabric-runtime-supervision` for provider readiness and reconcile health |
| owner route | resource ownership and next allowed Fabric action |
| evidence refs | K8s object refs, manifest checksums, DB operation refs, provider status refs |

## Contract Set

The first implementation should create these contracts under `contracts/`:

- `fabric-resource-catalog.schema.json`
- `fabric-lifecycle-ledger.schema.json`
- `fabric-event-envelope.schema.json`
- `fabric-human-gate.schema.json`
- `fabric-runtime-supervision.schema.json`
- `fabric-api.openapi.json`

The contracts must preserve the current OPL Cloud product rules:

- Basic Workspace is available: 2 CPU, 4GB memory, 10GB storage.
- Pro Workspace is available: 8 CPU, 16GB memory, 100GB storage.
- GPU Workspace is unavailable by default with reason `gpu_node_pool_not_verified`.
- Workspace URL path pattern is `/w/<workspaceId>`.
- Workspace token handling remains owned by Console; Fabric may mount or route token material only when Console provides it.
- Compute destroy does not destroy persistent storage.
- Storage destroy requires explicit confirmation or a recorded human gate.
- Backup deletion never deletes source storage, restored storage, or compute.
- Restore creates new billable storage from the Console/Ledger point of view, but Fabric only creates and verifies the restored PVC.

## Repository Layout

```text
contracts/
  fabric-api.openapi.json
  fabric-event-envelope.schema.json
  fabric-human-gate.schema.json
  fabric-lifecycle-ledger.schema.json
  fabric-resource-catalog.schema.json
  fabric-runtime-supervision.schema.json

docs/
  architecture.md
  decisions.md
  invariants.md
  status.md
  superpowers/specs/2026-07-03-opl-fabric-split-design.md

apps/fabric-api/
  cmd/fabric-api/main.go
  internal/catalog/
  internal/config/
  internal/domain/
  internal/evidence/
  internal/http/
  internal/k8s/
  internal/postgres/
  internal/service/

apps/fabric-console/
  index.html
  package.json
  tsconfig.json
  vite.config.ts
  src/

deploy/
  k8s/
  migrations/
```

`apps/fabric-api/internal/domain` contains pure Fabric domain types and state transitions. It must not import HTTP, PostgreSQL, or Kubernetes packages.

`apps/fabric-api/internal/k8s` owns all `client-go` interactions.

`apps/fabric-api/internal/postgres` owns schema migration application and repositories.

`apps/fabric-console` is an operator UI, not the commercial OPL Console.

## Backend API

The first HTTP API surface is:

```text
GET    /api/fabric/readiness
GET    /api/fabric/catalog
POST   /api/fabric/compute
POST   /api/fabric/storage
POST   /api/fabric/attachments
POST   /api/fabric/workspace-routes
POST   /api/fabric/compute/{id}:stop
POST   /api/fabric/compute/{id}:restart
DELETE /api/fabric/compute/{id}
DELETE /api/fabric/storage/{id}
POST   /api/fabric/storage-backups
POST   /api/fabric/storage-restores
GET    /api/fabric/resources/{id}/status
GET    /api/fabric/operations/{operationId}
```

All mutating calls accept:

- `idempotencyKey`
- `requestedBy`
- `correlationId`
- resource owner fields supplied by Console, such as `ownerAccountId`

All mutating responses include:

- `operationId`
- `status`
- `resourceRef`
- `evidenceRefs`
- `nextAllowedActions`

## Domain Model

Core entities:

- `ComputeResource`
- `StorageVolume`
- `StorageAttachment`
- `WorkspaceRoute`
- `StorageBackup`
- `FabricOperation`
- `FabricEvent`
- `FabricEvidenceRef`
- `HumanGate`

Core states:

- compute: `creating`, `running`, `stopping`, `stopped`, `restarting`, `destroying`, `destroyed`, `failed`, `cleanup_required`
- storage: `creating`, `available`, `attaching`, `attached`, `detaching`, `detached`, `destroying`, `destroyed`, `failed`, `cleanup_required`
- route: `configuring`, `ready`, `removing`, `removed`, `failed`
- backup: `creating`, `available`, `deleting`, `deleted`, `failed`
- operation: `accepted`, `dry_run`, `applying`, `verifying`, `succeeded`, `failed`, `blocked`, `needs_human_gate`

The domain layer enforces safety rules before provider calls:

- Destroying compute never implies destroying storage.
- Destroying storage is rejected unless confirmation or human gate evidence exists.
- Creating a WorkspaceRoute requires an attached compute target.
- Shared Ingress mutation requires optimistic concurrency and retry.
- Restore creates a new storage resource from a backup; it never writes over an existing PVC.

## PostgreSQL Persistence

PostgreSQL is the Fabric durable truth for resource state, operations, evidence refs, and idempotency.

Required tables:

- `compute_resources`
- `storage_volumes`
- `storage_attachments`
- `workspace_routes`
- `storage_backups`
- `fabric_operations`
- `fabric_events`
- `fabric_evidence_refs`
- `human_gates`
- `idempotency_keys`

Kubernetes is the execution plane and verification source, not the only state store.

## Kubernetes Provider

The first production provider is Tencent TKE through Kubernetes APIs.

The provider uses Go `client-go` for:

- Secret creation and update.
- PVC creation, deletion, and status reads.
- Deployment creation, scale, deletion, and rollout status.
- Service creation and endpoint status reads.
- Ingress route patching.
- Namespace, storage class, image pull secret, and ingress class readiness checks.

VolumeSnapshot support uses a dynamic client or typed external snapshot client. The provider must verify `readyToUse=true` before reporting backup success.

Provider actions must generate object manifests in memory, compute a SHA-256 digest, apply through the Kubernetes API, and record both manifest digest and observed object refs in evidence.

Shared Ingress updates must use Kubernetes resource version conflict handling. The provider retries conflicts and records a blocker if the route cannot be applied safely.

## Readiness

`GET /api/fabric/readiness` returns:

- `ready`
- `provider`
- `missingEnv`
- `missingKubernetesCapabilities`
- `resourceCatalog`
- `database`
- `kubernetes`
- `runtimeSupervision`
- `blockers`
- `repairHints`

Readiness checks include:

- PostgreSQL connection and migration version.
- Kubernetes API access.
- Namespace existence.
- StorageClass existence.
- ImagePullSecret existence when configured.
- IngressClass availability.
- Shared Ingress existence or creatability.
- VolumeSnapshot CRD and snapshot class availability.
- Workspace image and domain configuration.
- Basic and Pro package availability.
- GPU package remains unavailable unless GPU node pool verification is configured and passes.

## Operator Console

The React + TypeScript console is operator-facing.

It shows:

- readiness and blockers
- resource catalog
- compute resources
- storage volumes
- attachments
- routes
- backups
- operation history
- evidence refs
- human gates requiring action

It does not show:

- PI wallet balances
- billing ledger
- top-ups
- public pricing pages
- Lab Owner commercial workspace UX
- support tickets

The commercial OPL Console can later consume Fabric API responses and render Lab Owner workflows in its own repository.

## Migration From OPL-Cloud

Preserve from `OPL-Cloud`:

- Product contracts.
- Fabric catalog semantics.
- Workspace lifecycle safety rules.
- TKE resource behavior.
- Storage backup and restore rules.
- Production readiness expectations.
- Tests that assert product behavior, rewritten for Go/HTTP contracts.

Do not preserve:

- Node backend runtime.
- JS package import boundaries.
- `kubectl` subprocess execution as the normal provider path.
- JSON file store as production persistence.
- Commercial Console routes and UI pages inside Fabric.

## Verification Strategy

Required verification lanes:

- Go unit tests for domain state transitions and safety rules.
- Go repository tests against PostgreSQL.
- Go provider tests using Kubernetes fake clients.
- Contract schema validation for all JSON contracts.
- HTTP API tests for idempotency, blockers, and operation receipts.
- React TypeScript build and component tests for the operator console.
- `go test ./...`
- frontend typecheck and build
- `git diff --check`

Provider integration tests against a real cluster are separate from fast local tests and must be opt-in through environment variables.

## Open Decisions Resolved By This Spec

- `OPL-Fabric` is an independent Fabric service, not a full OPL Cloud rewrite.
- Contracts are the first implementation artifact.
- The backend is Go.
- PostgreSQL is required durable persistence.
- Kubernetes provider code uses `client-go`.
- The UI is an operator console, not the commercial OPL Console.
- `one-person-lab` is the development framework reference and contract style source, not an embedded runtime dependency.

## Out Of First Implementation Scope

- Public GA operations.
- External payment settlement.
- Full OPL Gateway product surface.
- Full standalone OPL Ledger service.
- Connector, environment, and agent marketplaces.
- GPU Workspace package exposure.
- Cross-region backup and restore.
- Application-level file versioning.
- In-place PVC restore.
