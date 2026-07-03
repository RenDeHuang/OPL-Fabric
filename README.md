# OPL Fabric

OPL Fabric is the Fabric control-plane service for OPL Cloud.

It owns resource catalog, provider readiness, compute lifecycle, storage lifecycle, storage attachment, Workspace routing, backup and restore mechanics, Kubernetes provider execution, and Fabric evidence.

It does not own OPL Console commercial flows, wallet and billing truth, OPL Ledger reconciliation, OPL Gateway AI routing, one-person-lab framework internals, or one-person-lab-app WebUI behavior.

## Stack

- Frontend: React + TypeScript
- Backend: Go
- DB: PostgreSQL
- Kubernetes: Go client-go

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

The manifest expects this placeholder Secret in the same namespace:

```bash
kubectl create secret generic opl-fabric-api-secrets \
  --from-literal=DATABASE_URL='postgres://user:password@postgres:5432/opl_fabric?sslmode=disable' \
  --from-literal=OPL_OPERATOR_TOKEN='replace-with-operator-token'
```

`DATABASE_URL` is consumed by the PostgreSQL store. `OPL_OPERATOR_TOKEN` is required by the HTTP server for Bearer authentication.

Future reconcile/status flows that read, watch, patch, or update Kubernetes objects must expand the Role deliberately with matching tests and review.
