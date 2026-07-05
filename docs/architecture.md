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
-> compute allocation
-> storage attachment
-> workspace entry
-> GET /api/fabric/workspaces/{id}
```

The decomposed storage, compute, attachment, and workspace-entry APIs remain published as lower-level resource routes. They are useful for Console advanced views, operator debugging, failure recovery, and evidence/provider refs consumed by OPL Ledger. Ledger records what happened; Fabric owns resource mutation.

## Controlled Live Boundary

Fabric is the cloud mutation boundary. The service records the requester, operation ID, resource kind, resource ID, provider refs, runtime refs, cluster ID, ComputePool/NodePool ID, and CVM/node refs so Console and Ledger can explain who created what, where it was created, and what must be retained or deleted.

ComputePool and ComputeAllocation are separate concepts. A ComputePool is a shared Tencent TKE NodePool for one package or provider instance type. A ComputeAllocation is the account-owned, workspace-exclusive CVM node assigned from that pool while the Workspace is active. Fabric models the normal product path as Workspace -> ComputeAllocation -> ComputePool, with retained storage mounted into the active runtime.

Delete is also operation-driven. Console must send an explicit confirmation request, Fabric creates a delete operation, the worker executes it, and storage retention policy decides whether PVC data is kept. The product rule remains: storage is durable, compute is rebuildable, and retained storage can be mounted by rebuilt compute.

The operator console is a React TypeScript UI for readiness, resource status, operation history, and evidence. It is not the commercial OPL Console.
