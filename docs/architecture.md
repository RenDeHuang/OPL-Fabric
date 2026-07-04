# Architecture

OPL Fabric is an independent service consumed by OPL Console through published HTTP APIs and JSON contracts.

The backend is split into pure domain rules, service orchestration, PostgreSQL persistence, Kubernetes provider execution, evidence recording, and HTTP transport.

The product create path is a single Workspace route for OPL Console:

```text
POST /api/fabric/workspaces
-> PostgreSQL transactional reservation
-> background worker lease
-> orchestrator
-> storage
-> compute
-> storage attachment
-> workspace entry
-> GET /api/fabric/workspaces/{id}
```

The decomposed storage, compute, attachment, and workspace-entry APIs remain published as lower-level resource routes. They are useful for Console advanced views, operator debugging, failure recovery, and evidence/provider refs consumed by OPL Ledger. Ledger records what happened; Fabric owns resource mutation.

## Controlled Live Boundary

Fabric is the cloud mutation boundary. The service records the requester, operation ID, resource kind, resource ID, provider refs, runtime refs, cluster ID, and NodePool ID so Console and Ledger can explain who created what, where it was created, and what must be retained or deleted.

Dedicated compute is not hard-coded as every Workspace owning a NodePool. It is an isolation mode. When a compute row requests `dedicated_nodepool` or the dedicated template pool, Fabric asks the Tencent Cloud Go SDK provider to create and verify a TKE NodePool before applying the Kubernetes Deployment/Service/Ingress path. Shared-pool compute skips this Tencent mutation and uses existing cluster capacity.

Delete is also operation-driven. Console must send an explicit confirmation request, Fabric creates a delete operation, the worker executes it, and storage retention policy decides whether PVC data is kept. The product rule remains: storage is durable, compute is rebuildable, and retained storage can be mounted by rebuilt compute.

The operator console is a React TypeScript UI for readiness, resource status, operation history, and evidence. It is not the commercial OPL Console.
