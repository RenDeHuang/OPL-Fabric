# Decisions

## 2026-07-03: OPL Fabric split

OPL Fabric is split as an independent Fabric service, not a full OPL Cloud rewrite.

The implementation is contract-first. The backend is Go, persistence is PostgreSQL, Kubernetes operations use client-go, and the frontend is React TypeScript.

one-person-lab is used as the development framework reference for contracts, lifecycle ledgers, evidence, human gates, readiness, and ownership boundaries.
