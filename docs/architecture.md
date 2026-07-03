# Architecture

OPL Fabric is an independent service consumed by OPL Console through published HTTP APIs and JSON contracts.

The backend is split into pure domain rules, service orchestration, PostgreSQL persistence, Kubernetes provider execution, evidence recording, and HTTP transport.

The operator console is a React TypeScript UI for readiness, resource status, operation history, and evidence. It is not the commercial OPL Console.
