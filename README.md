# OPL Fabric

OPL Fabric is the Fabric control-plane service for OPL Cloud.

It owns resource catalog, provider readiness, compute lifecycle, persistent storage lifecycle, storage attachment, Workspace entry routing, Kubernetes runtime execution, Tencent Cloud capacity execution, and Fabric evidence.

It does not own OPL Console commercial flows, wallet and billing truth, OPL Ledger reconciliation, OPL Gateway AI routing, one-person-lab framework internals, or one-person-lab-app WebUI behavior.

The product model is storage-first:

- Storage is the durable user asset.
- Compute is rebuildable execution capacity.
- Storage can be detached from one compute resource and reattached to another.
- A Workspace entry is derived from an attached storage volume, not from a bundled workspace package.

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

In a second shell, run the operator console. The Vite dev server proxies `/api` to `http://127.0.0.1:8787` and injects the Bearer token from its server-side `OPL_OPERATOR_TOKEN` environment variable, so the token is not exposed through a browser `VITE_` variable.

```bash
OPL_OPERATOR_TOKEN=dev-operator-token npm --prefix apps/fabric-console run dev
```

## Deployment Skeleton

`deploy/k8s/opl-fabric-api.yaml` contains a namespace-scoped skeleton for the Fabric API:

- `Deployment` and `Service` on port `8787`.
- `ServiceAccount`, `Role`, and `RoleBinding` with current minimal client-go permissions: create/delete Deployments and create Services in the namespace.
- Default image `opl-fabric-api:local`; replace it with a registry image in your deployment pipeline or overlay.
- `OPL_K8S_NAMESPACE` populated from the pod metadata namespace.
- Workspace defaults matching backend config: `OPL_WORKSPACE_IMAGE`, `OPL_WORKSPACE_DOMAIN`, and `OPL_WORKSPACE_STORAGE_CLASS`.
- Tencent capacity defaults are placeholders only: `TENCENT_TKE_REGION`, `TENCENT_DEPLOY_CLUSTER_ID`, TCR refs, and hourly node pool charge type. Real mutation credentials must come from Secret keys.

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
