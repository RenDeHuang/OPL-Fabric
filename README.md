# OPL Fabric

OPL Fabric is the Fabric control-plane service for OPL Cloud.

It owns resource catalog, provider readiness, compute lifecycle, persistent storage lifecycle, storage attachment, Workspace entry routing, Kubernetes runtime execution, Tencent Cloud capacity execution, and Fabric evidence.

It does not own OPL Console commercial flows, wallet and billing truth, OPL Ledger reconciliation, OPL Gateway AI routing, one-person-lab framework internals, or one-person-lab-app WebUI behavior.

The product model is a recoverable personal Workspace:

- Storage is the durable user asset.
- A ComputePool is a package or instance-type level Tencent TKE NodePool.
- A ComputeAllocation is one account-owned CVM node from a ComputePool while a Workspace is active.
- Compute is rebuildable execution capacity: it can be destroyed and recreated without deleting storage.
- Storage can be detached from one ComputeAllocation and reattached to another.
- A Workspace entry is the stable URL for the current storage attachment and one-person-lab-app runtime.

## Stack

- Frontend: React + TypeScript
- Backend: Go
- DB: PostgreSQL
- Kubernetes: Go client-go
- Cloud Provider: Tencent Cloud Go SDK
- API Contract: OpenAPI + JSON Schema
- Runtime Config: `config/` + `OPL_FABRIC_CONFIG_DIR`

Long-term runtime dependencies explicitly forbidden:

- no `kubectl` shell-out for normal runtime
- no `tccli` shell-out for normal runtime
- no JavaScript provider runtime

Kubernetes runtime truth comes from Kubernetes API/client-go. Tencent Cloud capacity truth comes from Tencent Cloud Go SDK. Fabric durable truth comes from PostgreSQL.

## Configuration

The default configuration catalog lives in `config/`.

Set `OPL_FABRIC_CONFIG_DIR=/path/to/config` when you need to use another path with the same file names.

Real secrets must not be committed. Use Kubernetes Secrets, external secret refs, or ignored local env files based on `config/fabric.env.example`.

## Local Verification

The root verification command skips workspaces that have not been created yet and runs their checks once they exist.

```bash
npm test
```

Focused backend verification:

```bash
npm run test:go
```

Focused operator console verification:

```bash
npm run test:console
```

## Local Development

The Fabric API listens on port `8787` by default and requires `Authorization: Bearer $OPL_OPERATOR_TOKEN` for all HTTP routes.

```bash
cd apps/fabric-api
OPL_OPERATOR_TOKEN=dev-operator-token go run ./cmd/fabric-api
```

Mutating reservation endpoints require `DATABASE_URL`; when it is configured the API opens PostgreSQL and runs the embedded migration before serving.

The Console-facing product mainline is `POST /api/fabric/workspaces`. It reserves storage, compute allocation intent, storage attachment, Workspace entry, and a Fabric operation in one PostgreSQL transaction, then returns an operation receipt. When `OPL_FABRIC_WORKER_ENABLED=true`, the background worker leases the accepted workspace operation and executes the chain through the orchestrator:

```text
Workspace create -> storage -> compute allocation -> attach -> entry -> running Workspace URL
```

The single-resource APIs remain available as lower-level operational routes:

- `POST /api/fabric/storage-volumes` and `GET /api/fabric/storage-volumes/{id}`
- `POST /api/fabric/compute-allocations` and `GET /api/fabric/compute-allocations/{id}`
- `POST /api/fabric/storage-attachments` and `GET /api/fabric/storage-attachments/{id}`
- `POST /api/fabric/workspace-entries` and `GET /api/fabric/workspace-entries/{id}`

OPL Console should use the workspace route as the normal create path and use the single-resource routes for advanced resource views, operator debugging, and failure recovery. OPL Ledger should receive operation, evidence, and provider refs from Fabric; it should not create cloud resources itself.

The worker is disabled by default in local env examples and the Kubernetes skeleton. Enabling it in staging requires verified PostgreSQL, in-cluster Kubernetes access, storage class, ingress class, Workspace image, pull secrets, and explicit live readiness review.

## Controlled Live Resource Mutation

Fabric is the only service that should create or delete cloud resources for a Workspace. OPL Console calls Fabric APIs and polls operation/resource status. OPL Ledger records Fabric operation IDs, provider refs, evidence refs, cluster IDs, ComputePool/NodePool IDs, CVM/node refs, requester identity, and timestamps; it does not mutate cloud resources directly.

Live resource mutation is intentionally gated:

- `POST /api/fabric/workspaces` is the product create path. The worker executes storage -> compute -> attach -> entry.
- ComputePools are shared by specification or provider instance type. They are not created per Workspace as the normal product model.
- Each active Workspace receives a workspace-exclusive ComputeAllocation: one CVM node from the matching ComputePool, running one one-person-lab-app container with that Workspace storage mounted.
- Destroying compute releases the active ComputeAllocation. It must not delete the retained storage or the Workspace URL record.
- Delete routes require confirmation and create accepted operations. Runtime deletion runs through Fabric so provider refs and retained storage policy stay auditable.
- `OPL_TKE_ALLOW_NODEPOOL_MUTATION=false` is the safe default. Staging or production must explicitly set it to `true`, enable the worker, and pass the staging gate before live ComputePool/NodePool mutation is allowed.

In a second shell, run the operator console. The Vite dev server proxies `/api` to `http://127.0.0.1:8787` and injects the Bearer token from its server-side `OPL_OPERATOR_TOKEN` environment variable, so the token is not exposed through a browser `VITE_` variable.

```bash
OPL_OPERATOR_TOKEN=dev-operator-token npm --prefix apps/fabric-console run dev
```

## Deployment Skeleton

`deploy/k8s/opl-fabric-api.yaml` contains a namespace-scoped skeleton for the Fabric API:

- `Namespace`, `Deployment`, and `Service` on port `8787`.
- `ServiceAccount`, `Role`, and `RoleBinding` with namespace-scoped client-go permissions for Deployments, Services, PVCs, Secrets, and Ingresses used by the worker path.
- Default image `opl-fabric-api:local`; replace it with a registry image in your deployment pipeline or overlay.
- `OPL_K8S_NAMESPACE` populated from the pod metadata namespace.
- Workspace defaults matching backend config: `OPL_WORKSPACE_IMAGE`, `OPL_WORKSPACE_DOMAIN`, and `OPL_WORKSPACE_STORAGE_CLASS`.
- Worker defaults are present but disabled through `OPL_FABRIC_WORKER_ENABLED=false`.
- Tencent capacity defaults are placeholders only: `TENCENT_TKE_REGION`, `TENCENT_DEPLOY_CLUSTER_ID`, TCR refs, CVM subnet/security group IDs, system disk defaults, and hourly ComputePool/NodePool charge type. Real mutation credentials must come from Secret keys.
- ComputePool/NodePool mutation is explicitly disabled by `OPL_TKE_ALLOW_NODEPOOL_MUTATION=false`. Staging/prod overlays must opt in only after the gate reports `ready_for_live`.

The manifest expects this placeholder Secret in the same namespace:

```bash
kubectl create secret generic opl-fabric-api-secrets \
  --from-literal=DATABASE_URL='postgres://user:password@postgres:5432/opl_fabric?sslmode=disable' \
  --from-literal=OPL_OPERATOR_TOKEN='replace-with-operator-token' \
  --from-literal=TENCENT_MUTATION_SECRET_ID='replace-with-tencent-secret-id' \
  --from-literal=TENCENT_MUTATION_SECRET_KEY='replace-with-tencent-secret-key'
```

`DATABASE_URL` is consumed by the PostgreSQL store. `OPL_OPERATOR_TOKEN` is required by the HTTP server for Bearer authentication. Tencent mutation credentials are for the Tencent Cloud Go SDK capacity boundary.

Future reconcile/status flows that read, watch, patch, or update Kubernetes objects must expand the Role deliberately with matching tests and review.
