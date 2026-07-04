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

The operator console is a React TypeScript UI for readiness, resource status, operation history, and evidence. It is not the commercial OPL Console.
