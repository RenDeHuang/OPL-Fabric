# OPL Fabric Config

This directory is the default OPL-Fabric configuration root.

Set `OPL_FABRIC_CONFIG_DIR=/path/to/config` to use another directory with the same file names.

Committed files are examples and key manifests only. Do not commit real passwords, API keys, kubeconfig contents, TCR credentials, or PostgreSQL passwords.

## Files

- `fabric.env.example`: local and production environment key template for the Fabric API.
- `production-manifest.example.json`: secretRef/value manifest for production handoff.
- `config.keys.json`: machine-readable list of supported keys, owners, and sensitivity.
- `tke/workspace-runtime.env.example`: workspace pod/runtime defaults derived from medopl-3.
- `tke/readiness-checks.json`: readiness checks OPL-Fabric should perform before production traffic.
- `tke/secret-refs.example.json`: Kubernetes Secret and external secret reference names.
- `sources/medopl-3-baseline.md`: source baseline and migration notes from medopl-3 and the active OPL Cloud baseline.
- `sources/medopl-3/`: copied medopl-3 example config and deployment contract snapshots.
- `sources/opl-cloud-126e6bf/`: copied active OPL Cloud contract snapshots.
- `sources/opl-cloud-126e6bf-production-api-baseline.md`: notes on the active OPL Cloud API/resource baseline.

## Runtime Loading

The Go process still reads final values from environment variables. This directory defines the canonical names, defaults, and secret references used by local env files, Kubernetes manifests, and future render tools.

The runtime stack is fixed: Kubernetes operations use Go client-go, Tencent capacity operations use Tencent Cloud Go SDK, and Fabric state uses PostgreSQL. `kubectl`, `tccli`, and JavaScript provider runtimes are reference-only patterns and must not become normal runtime dependencies.
