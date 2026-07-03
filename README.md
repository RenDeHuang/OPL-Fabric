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
