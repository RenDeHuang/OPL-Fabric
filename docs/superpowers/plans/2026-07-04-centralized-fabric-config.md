# Centralized Fabric Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Centralize all OPL-Fabric configurable runtime, TKE, workspace, Codex, readiness, and secret-reference inputs under one configurable directory that can later be moved by changing `OPL_FABRIC_CONFIG_DIR`.

**Architecture:** `config/` becomes the default configuration root. The repository stores examples, schemas, key manifests, and source notes only; real secrets stay outside git and are referenced by path or `secretRef`. Go code keeps environment variables as the final runtime source, but gains a small config-root loader so generated deployment manifests and local runs can point at a different config directory.

**Tech Stack:** Go, React + TypeScript, PostgreSQL, Kubernetes `client-go`, JSON, dotenv-style env examples, Kubernetes YAML.

---

## Current Source Baselines

Use these source inputs:

- OPL-Fabric current `main`: `7cca8abd1087a35479718f53fa015fedfb10e7e1`.
- OPL-Cloud pinned split baseline: `RenDeHuang/OPL-Cloud@2985bfdaa592a0644da5fdb0c11a877785a85155`.
- OPL-Cloud latest observed in `/home/dev/medopl-3`: `d2c7474deb6deb39daf81232f563a5f39c0fdd16`.
- `medopl-3` path: `/home/dev/medopl-3`.

The relevant `medopl-3` files are:

- `/home/dev/medopl-3/deploy/tke/opl-cloud-production.env.example`
- `/home/dev/medopl-3/deploy/production-manifest.example.json`
- `/home/dev/medopl-3/packages/console/src/production-manifest.js`
- `/home/dev/medopl-3/packages/console/src/production-readiness.js`
- `/home/dev/medopl-3/packages/fabric/src/runtime-providers/tencent-tke.js`
- `/home/dev/medopl-3/docs/runtime/tke-production-deployment.md`
- `/home/dev/medopl-3/docs/runtime/production-runbook.md`

Do not copy real values from `.runtime/`, `.env`, or local secret files. Only copy public examples, key names, default paths, and secret reference shapes.

## Target File Structure

Create this structure:

```text
config/
  README.md
  fabric.env.example
  production-manifest.example.json
  config.keys.json
  tke/
    workspace-runtime.env.example
    readiness-checks.json
    secret-refs.example.json
  sources/
    medopl-3-baseline.md
```

Modify these existing files:

```text
apps/fabric-api/internal/config/config.go
apps/fabric-api/internal/config/config_test.go
apps/fabric-api/internal/service/service.go
apps/fabric-api/internal/service/service_test.go
apps/fabric-api/internal/k8s/provider.go
apps/fabric-api/internal/k8s/provider_test.go
deploy/k8s/opl-fabric-api.yaml
README.md
docs/status.md
docs/decisions.md
```

Do not modify:

```text
/home/dev/medopl-3/**
```

## Config Ownership Rules

- `config/` is the default editable config path.
- `OPL_FABRIC_CONFIG_DIR` can point to another path later.
- Committed files contain examples and non-secret defaults only.
- Real secret material must be passed through environment variables, Kubernetes Secrets, or external files referenced by secret refs.
- OPL-Fabric owns Fabric keys only. Console billing/auth keys from `medopl-3` may be documented as upstream context but should not become required Fabric runtime keys.

## Configuration Keys To Centralize

Fabric runtime:

```text
PORT
DATABASE_URL
OPL_OPERATOR_TOKEN
OPL_FABRIC_CONFIG_DIR
```

Workspace runtime:

```text
OPL_WORKSPACE_IMAGE
OPL_WORKSPACE_DOMAIN
OPL_WORKSPACE_WEBUI_PORT
OPL_WORKSPACE_DATA_DIR
OPL_WORKSPACE_PROJECTS_DIR
OPL_WORKSPACE_STORAGE_CLASS
OPL_WORKSPACE_VOLUME_SNAPSHOT_CLASS
OPL_WORKSPACE_NODE_SELECTOR_KEY
OPL_WORKSPACE_NODE_SELECTOR_VALUE
```

TKE provider:

```text
OPL_K8S_NAMESPACE
OPL_INGRESS_CLASS
OPL_IMAGE_PULL_SECRET_NAME
TENCENT_DEPLOY_KUBECONFIG_REF
TENCENT_DEPLOY_CLUSTER_ID
TENCENT_TCR_REGISTRY
TENCENT_TCR_NAMESPACE
TENCENT_TCR_REGION
```

TLS and routing:

```text
OPL_TLS_CERT_ID
OPL_WORKSPACE_TLS_SECRET_NAME
OPL_WORKSPACE_TLS_CERT_ID
```

Codex workspace bootstrap:

```text
OPL_CODEX_MODEL
OPL_CODEX_REASONING_EFFORT
OPL_CODEX_BASE_URL
OPL_CODEX_API_KEY
OPL_CODEX_MODEL_PROVIDER
OPL_CODEX_PROVIDER_NAME
CODEX_HOME
```

## Task 1: Add Central Config Directory

**Files:**

- Create: `config/README.md`
- Create: `config/fabric.env.example`
- Create: `config/production-manifest.example.json`
- Create: `config/config.keys.json`
- Create: `config/tke/workspace-runtime.env.example`
- Create: `config/tke/readiness-checks.json`
- Create: `config/tke/secret-refs.example.json`
- Create: `config/sources/medopl-3-baseline.md`

- [ ] **Step 1: Create the config README**

Write `config/README.md`:

```markdown
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
- `sources/medopl-3-baseline.md`: source baseline and migration notes from medopl-3.

## Runtime Loading

The Go process still reads final values from environment variables. This directory defines the canonical names, defaults, and secret references used by local env files, Kubernetes manifests, and future render tools.
```

- [ ] **Step 2: Create the Fabric env example**

Write `config/fabric.env.example`:

```dotenv
# OPL Fabric API.
PORT=8787
OPL_FABRIC_CONFIG_DIR=./config
DATABASE_URL=
OPL_OPERATOR_TOKEN=

# Workspace catalog and runtime.
OPL_WORKSPACE_IMAGE=ghcr.io/gaofeng21cn/one-person-lab-app:latest
OPL_WORKSPACE_DOMAIN=workspace.medopl.cn
OPL_WORKSPACE_WEBUI_PORT=3000
OPL_WORKSPACE_DATA_DIR=/data
OPL_WORKSPACE_PROJECTS_DIR=/projects
OPL_WORKSPACE_STORAGE_CLASS=cbs
OPL_WORKSPACE_VOLUME_SNAPSHOT_CLASS=
OPL_WORKSPACE_NODE_SELECTOR_KEY=medopl.cn/workload
OPL_WORKSPACE_NODE_SELECTOR_VALUE=medopl

# Tencent TKE provider.
OPL_K8S_NAMESPACE=opl-cloud
OPL_INGRESS_CLASS=qcloud
OPL_IMAGE_PULL_SECRET_NAME=tcr-pull-secret
TENCENT_DEPLOY_KUBECONFIG_REF=
TENCENT_DEPLOY_CLUSTER_ID=
TENCENT_TCR_REGISTRY=
TENCENT_TCR_NAMESPACE=
TENCENT_TCR_REGION=

# TKE TLS and route configuration.
OPL_TLS_CERT_ID=
OPL_WORKSPACE_TLS_SECRET_NAME=opl-cloud-workspace-medopl-cn-tls
OPL_WORKSPACE_TLS_CERT_ID=

# Workspace Codex bootstrap.
OPL_CODEX_MODEL=gpt-5.5
OPL_CODEX_REASONING_EFFORT=xhigh
OPL_CODEX_BASE_URL=https://gflabtoken.cn/v1
OPL_CODEX_API_KEY=
OPL_CODEX_MODEL_PROVIDER=gflabtoken
OPL_CODEX_PROVIDER_NAME=gflabtoken
CODEX_HOME=/data/codex
```

- [ ] **Step 3: Create the production manifest example**

Write `config/production-manifest.example.json`:

```json
{
  "env": {
    "PORT": { "value": "8787" },
    "OPL_FABRIC_CONFIG_DIR": { "value": "/etc/opl-fabric/config" },
    "DATABASE_URL": { "secretRef": "opl-fabric/database-url" },
    "OPL_OPERATOR_TOKEN": { "secretRef": "opl-fabric/operator-token" },
    "OPL_WORKSPACE_DOMAIN": { "value": "workspace.medopl.cn" },
    "OPL_WORKSPACE_IMAGE": { "value": "registry.example.com/opl/one-person-lab-app:2026-07-01" },
    "OPL_WORKSPACE_WEBUI_PORT": { "value": "3000" },
    "OPL_WORKSPACE_DATA_DIR": { "value": "/data" },
    "OPL_WORKSPACE_PROJECTS_DIR": { "value": "/projects" },
    "OPL_WORKSPACE_STORAGE_CLASS": { "value": "cbs" },
    "OPL_WORKSPACE_VOLUME_SNAPSHOT_CLASS": { "value": "" },
    "OPL_WORKSPACE_NODE_SELECTOR_KEY": { "value": "medopl.cn/workload" },
    "OPL_WORKSPACE_NODE_SELECTOR_VALUE": { "value": "medopl" },
    "OPL_K8S_NAMESPACE": { "value": "opl-cloud" },
    "OPL_INGRESS_CLASS": { "value": "qcloud" },
    "OPL_IMAGE_PULL_SECRET_NAME": { "value": "tcr-pull-secret" },
    "OPL_TLS_CERT_ID": { "secretRef": "opl-fabric/tls-cert-id" },
    "OPL_WORKSPACE_TLS_SECRET_NAME": { "value": "opl-cloud-workspace-medopl-cn-tls" },
    "OPL_WORKSPACE_TLS_CERT_ID": { "secretRef": "opl-fabric/workspace-tls-cert-id" },
    "OPL_CODEX_MODEL": { "value": "gpt-5.5" },
    "OPL_CODEX_REASONING_EFFORT": { "value": "xhigh" },
    "OPL_CODEX_BASE_URL": { "value": "https://gflabtoken.cn/v1" },
    "OPL_CODEX_API_KEY": { "secretRef": "opl-fabric/codex-api-key" },
    "OPL_CODEX_MODEL_PROVIDER": { "value": "gflabtoken" },
    "OPL_CODEX_PROVIDER_NAME": { "value": "gflabtoken" },
    "CODEX_HOME": { "value": "/data/codex" },
    "TENCENT_DEPLOY_KUBECONFIG_REF": { "secretRef": "opl-fabric/tencent-deploy-kubeconfig-ref" },
    "TENCENT_DEPLOY_CLUSTER_ID": { "value": "cls-xxxxxxxx" },
    "TENCENT_TCR_REGISTRY": { "value": "registry.example.com" },
    "TENCENT_TCR_NAMESPACE": { "value": "opl" },
    "TENCENT_TCR_REGION": { "value": "ap-guangzhou" }
  }
}
```

- [ ] **Step 4: Create the machine-readable key manifest**

Write `config/config.keys.json`:

```json
{
  "schemaVersion": 1,
  "owner": "OPL Fabric",
  "defaultConfigDir": "config",
  "overrideEnv": "OPL_FABRIC_CONFIG_DIR",
  "keys": [
    { "name": "PORT", "owner": "fabric-api", "required": false, "secret": false, "default": "8787" },
    { "name": "DATABASE_URL", "owner": "fabric-api", "required": true, "secret": true },
    { "name": "OPL_OPERATOR_TOKEN", "owner": "fabric-api", "required": true, "secret": true },
    { "name": "OPL_WORKSPACE_IMAGE", "owner": "workspace-runtime", "required": true, "secret": false },
    { "name": "OPL_WORKSPACE_DOMAIN", "owner": "workspace-runtime", "required": true, "secret": false },
    { "name": "OPL_WORKSPACE_WEBUI_PORT", "owner": "workspace-runtime", "required": false, "secret": false, "default": "3000" },
    { "name": "OPL_WORKSPACE_DATA_DIR", "owner": "workspace-runtime", "required": false, "secret": false, "default": "/data" },
    { "name": "OPL_WORKSPACE_PROJECTS_DIR", "owner": "workspace-runtime", "required": false, "secret": false, "default": "/projects" },
    { "name": "OPL_WORKSPACE_STORAGE_CLASS", "owner": "tke-provider", "required": true, "secret": false },
    { "name": "OPL_WORKSPACE_VOLUME_SNAPSHOT_CLASS", "owner": "tke-provider", "required": false, "secret": false },
    { "name": "OPL_WORKSPACE_NODE_SELECTOR_KEY", "owner": "tke-provider", "required": false, "secret": false },
    { "name": "OPL_WORKSPACE_NODE_SELECTOR_VALUE", "owner": "tke-provider", "required": false, "secret": false },
    { "name": "OPL_K8S_NAMESPACE", "owner": "tke-provider", "required": true, "secret": false },
    { "name": "OPL_INGRESS_CLASS", "owner": "tke-provider", "required": true, "secret": false },
    { "name": "OPL_IMAGE_PULL_SECRET_NAME", "owner": "tke-provider", "required": true, "secret": false },
    { "name": "TENCENT_DEPLOY_KUBECONFIG_REF", "owner": "tke-provider", "required": true, "secret": true },
    { "name": "TENCENT_DEPLOY_CLUSTER_ID", "owner": "tencent", "required": true, "secret": false },
    { "name": "TENCENT_TCR_REGISTRY", "owner": "tencent", "required": true, "secret": false },
    { "name": "TENCENT_TCR_NAMESPACE", "owner": "tencent", "required": true, "secret": false },
    { "name": "TENCENT_TCR_REGION", "owner": "tencent", "required": true, "secret": false },
    { "name": "OPL_TLS_CERT_ID", "owner": "tke-provider", "required": false, "secret": true },
    { "name": "OPL_WORKSPACE_TLS_SECRET_NAME", "owner": "tke-provider", "required": false, "secret": false },
    { "name": "OPL_WORKSPACE_TLS_CERT_ID", "owner": "tke-provider", "required": false, "secret": true },
    { "name": "OPL_CODEX_MODEL", "owner": "workspace-codex", "required": false, "secret": false, "default": "gpt-5.5" },
    { "name": "OPL_CODEX_REASONING_EFFORT", "owner": "workspace-codex", "required": false, "secret": false, "default": "xhigh" },
    { "name": "OPL_CODEX_BASE_URL", "owner": "workspace-codex", "required": false, "secret": false, "default": "https://gflabtoken.cn/v1" },
    { "name": "OPL_CODEX_API_KEY", "owner": "workspace-codex", "required": false, "secret": true },
    { "name": "OPL_CODEX_MODEL_PROVIDER", "owner": "workspace-codex", "required": false, "secret": false, "default": "gflabtoken" },
    { "name": "OPL_CODEX_PROVIDER_NAME", "owner": "workspace-codex", "required": false, "secret": false, "default": "gflabtoken" },
    { "name": "CODEX_HOME", "owner": "workspace-codex", "required": false, "secret": false, "default": "/data/codex" }
  ]
}
```

- [ ] **Step 5: Create TKE workspace runtime example**

Write `config/tke/workspace-runtime.env.example`:

```dotenv
OPL_WORKSPACE_WEBUI_PORT=3000
OPL_WORKSPACE_DATA_DIR=/data
OPL_WORKSPACE_PROJECTS_DIR=/projects
OPL_PROJECTS_DIR=/projects
OPL_WEBUI_AUTH_MODE=none
OPL_WORKSPACE_ROOT=/projects
CODEX_HOME=/data/codex
```

- [ ] **Step 6: Create readiness checks config**

Write `config/tke/readiness-checks.json`:

```json
{
  "schemaVersion": 1,
  "provider": "tencent-tke",
  "requiredEnv": [
    "DATABASE_URL",
    "OPL_OPERATOR_TOKEN",
    "OPL_WORKSPACE_IMAGE",
    "OPL_WORKSPACE_DOMAIN",
    "OPL_K8S_NAMESPACE",
    "OPL_INGRESS_CLASS",
    "OPL_IMAGE_PULL_SECRET_NAME",
    "OPL_WORKSPACE_STORAGE_CLASS",
    "TENCENT_DEPLOY_KUBECONFIG_REF",
    "TENCENT_DEPLOY_CLUSTER_ID",
    "TENCENT_TCR_REGISTRY",
    "TENCENT_TCR_NAMESPACE",
    "TENCENT_TCR_REGION"
  ],
  "optionalEnv": [
    "OPL_WORKSPACE_VOLUME_SNAPSHOT_CLASS",
    "OPL_WORKSPACE_NODE_SELECTOR_KEY",
    "OPL_WORKSPACE_NODE_SELECTOR_VALUE",
    "OPL_TLS_CERT_ID",
    "OPL_WORKSPACE_TLS_SECRET_NAME",
    "OPL_WORKSPACE_TLS_CERT_ID",
    "OPL_CODEX_MODEL",
    "OPL_CODEX_REASONING_EFFORT",
    "OPL_CODEX_BASE_URL",
    "OPL_CODEX_API_KEY",
    "OPL_CODEX_MODEL_PROVIDER",
    "OPL_CODEX_PROVIDER_NAME",
    "CODEX_HOME"
  ],
  "clusterChecks": [
    "postgres_connection",
    "postgres_migration_version",
    "kubernetes_api_access",
    "namespace_exists",
    "storage_class_exists",
    "image_pull_secret_exists",
    "ingress_class_exists",
    "shared_ingress_exists_or_creatable",
    "volume_snapshot_crd_exists",
    "volume_snapshot_class_exists_when_configured"
  ],
  "workspaceContract": {
    "webuiPort": 3000,
    "dataDir": "/data",
    "projectsDir": "/projects",
    "codexHome": "/data/codex"
  }
}
```

- [ ] **Step 7: Create secret refs example**

Write `config/tke/secret-refs.example.json`:

```json
{
  "schemaVersion": 1,
  "kubernetesSecret": "opl-fabric-api-secrets",
  "keys": {
    "DATABASE_URL": "DATABASE_URL",
    "OPL_OPERATOR_TOKEN": "OPL_OPERATOR_TOKEN",
    "OPL_CODEX_API_KEY": "OPL_CODEX_API_KEY",
    "TENCENT_DEPLOY_KUBECONFIG_REF": "TENCENT_DEPLOY_KUBECONFIG_REF",
    "OPL_TLS_CERT_ID": "OPL_TLS_CERT_ID",
    "OPL_WORKSPACE_TLS_CERT_ID": "OPL_WORKSPACE_TLS_CERT_ID"
  },
  "externalSecretRefs": {
    "DATABASE_URL": "opl-fabric/database-url",
    "OPL_OPERATOR_TOKEN": "opl-fabric/operator-token",
    "OPL_CODEX_API_KEY": "opl-fabric/codex-api-key",
    "TENCENT_DEPLOY_KUBECONFIG_REF": "opl-fabric/tencent-deploy-kubeconfig-ref",
    "OPL_TLS_CERT_ID": "opl-fabric/tls-cert-id",
    "OPL_WORKSPACE_TLS_CERT_ID": "opl-fabric/workspace-tls-cert-id"
  }
}
```

- [ ] **Step 8: Record medopl-3 source baseline**

Write `config/sources/medopl-3-baseline.md`:

```markdown
# medopl-3 Baseline

Source path: `/home/dev/medopl-3`

Observed commit:

- `d2c7474 Bootstrap Codex config for TKE workspaces`
- Parent feature commit: `25403cc Inject Codex provider settings into TKE workspaces`

Files read:

- `deploy/tke/opl-cloud-production.env.example`
- `deploy/production-manifest.example.json`
- `packages/console/src/production-manifest.js`
- `packages/console/src/production-readiness.js`
- `packages/fabric/src/runtime-providers/tencent-tke.js`
- `docs/runtime/tke-production-deployment.md`
- `docs/runtime/production-runbook.md`

Imported concepts:

- Tencent TKE is the production provider.
- TCR image references must match the configured registry.
- Workspace runtime exposes port `3000`.
- Workspace persistent data uses `/data`.
- Workspace projects use `/projects`.
- Codex config is bootstrapped under `/data/codex`.
- Sensitive values must use secret refs or external secret files.
- VolumeSnapshot support depends on `snapshot.storage.k8s.io`.

Not imported:

- Commercial Console users and billing runtime keys.
- Real secrets from `.runtime`, `.env`, or `/home/dev/.secrets`.
- JavaScript `kubectl` execution as a runtime implementation pattern.
```

- [ ] **Step 9: Verify config files are valid**

Run:

```bash
node -e 'for (const f of ["config/production-manifest.example.json","config/config.keys.json","config/tke/readiness-checks.json","config/tke/secret-refs.example.json"]) JSON.parse(require("fs").readFileSync(f,"utf8")); console.log("config json ok")'
```

Expected output:

```text
config json ok
```

- [ ] **Step 10: Commit**

Run:

```bash
git add config
git commit -m "docs: add centralized fabric config catalog"
```

## Task 2: Add Go Config Root and New Key Fields

**Files:**

- Modify: `apps/fabric-api/internal/config/config.go`
- Create: `apps/fabric-api/internal/config/config_test.go`

- [ ] **Step 1: Write config tests**

Create `apps/fabric-api/internal/config/config_test.go`:

```go
package config

import "testing"

func TestLoadReadsFabricConfigDirAndWorkspaceDefaults(t *testing.T) {
	t.Setenv("OPL_FABRIC_CONFIG_DIR", "/tmp/opl-fabric-config")
	t.Setenv("OPL_WORKSPACE_WEBUI_PORT", "3001")
	t.Setenv("OPL_WORKSPACE_DATA_DIR", "/workspace-data")
	t.Setenv("OPL_WORKSPACE_PROJECTS_DIR", "/workspace-projects")
	t.Setenv("OPL_WORKSPACE_VOLUME_SNAPSHOT_CLASS", "cbs-snap")
	t.Setenv("OPL_WORKSPACE_NODE_SELECTOR_KEY", "medopl.cn/workload")
	t.Setenv("OPL_WORKSPACE_NODE_SELECTOR_VALUE", "medopl")
	t.Setenv("OPL_INGRESS_CLASS", "qcloud")
	t.Setenv("OPL_IMAGE_PULL_SECRET_NAME", "tcr-pull-secret")
	t.Setenv("OPL_CODEX_MODEL", "gpt-5.5")
	t.Setenv("OPL_CODEX_REASONING_EFFORT", "xhigh")
	t.Setenv("OPL_CODEX_BASE_URL", "https://gflabtoken.cn/v1")
	t.Setenv("OPL_CODEX_API_KEY", "secret")
	t.Setenv("OPL_CODEX_MODEL_PROVIDER", "gflabtoken")
	t.Setenv("OPL_CODEX_PROVIDER_NAME", "gflabtoken")
	t.Setenv("CODEX_HOME", "/data/codex")

	cfg := Load()

	if cfg.ConfigDir != "/tmp/opl-fabric-config" {
		t.Fatalf("ConfigDir = %q", cfg.ConfigDir)
	}
	if cfg.WorkspaceWebUIPort != "3001" {
		t.Fatalf("WorkspaceWebUIPort = %q", cfg.WorkspaceWebUIPort)
	}
	if cfg.WorkspaceVolumeSnapshotClass != "cbs-snap" {
		t.Fatalf("WorkspaceVolumeSnapshotClass = %q", cfg.WorkspaceVolumeSnapshotClass)
	}
	if cfg.CodexAPIKey != "secret" {
		t.Fatalf("CodexAPIKey not loaded")
	}
}

func TestLoadUsesProductionCompatibleDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ConfigDir != "config" {
		t.Fatalf("ConfigDir = %q", cfg.ConfigDir)
	}
	if cfg.WorkspaceWebUIPort != "3000" {
		t.Fatalf("WorkspaceWebUIPort = %q", cfg.WorkspaceWebUIPort)
	}
	if cfg.WorkspaceDataDir != "/data" {
		t.Fatalf("WorkspaceDataDir = %q", cfg.WorkspaceDataDir)
	}
	if cfg.WorkspaceProjectsDir != "/projects" {
		t.Fatalf("WorkspaceProjectsDir = %q", cfg.WorkspaceProjectsDir)
	}
	if cfg.CodexHome != "/data/codex" {
		t.Fatalf("CodexHome = %q", cfg.CodexHome)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd apps/fabric-api && go test ./internal/config
```

Expected: FAIL because `ConfigDir`, workspace runtime, TKE, and Codex fields are not defined yet.

- [ ] **Step 3: Extend Go config**

Modify `apps/fabric-api/internal/config/config.go` to:

```go
package config

import "os"

type Config struct {
	Port                         string
	ConfigDir                    string
	DatabaseURL                  string
	WorkspaceImage               string
	WorkspaceDomain              string
	WorkspaceWebUIPort           string
	WorkspaceDataDir             string
	WorkspaceProjectsDir         string
	StorageClass                 string
	WorkspaceVolumeSnapshotClass string
	WorkspaceNodeSelectorKey     string
	WorkspaceNodeSelectorValue   string
	KubernetesNamespace          string
	IngressClass                 string
	ImagePullSecretName          string
	OperatorToken                string
	CodexModel                   string
	CodexReasoningEffort         string
	CodexBaseURL                 string
	CodexAPIKey                  string
	CodexModelProvider           string
	CodexProviderName            string
	CodexHome                    string
}

func Load() Config {
	return Config{
		Port:                         env("PORT", "8787"),
		ConfigDir:                    env("OPL_FABRIC_CONFIG_DIR", "config"),
		DatabaseURL:                  os.Getenv("DATABASE_URL"),
		WorkspaceImage:               env("OPL_WORKSPACE_IMAGE", "ghcr.io/gaofeng21cn/one-person-lab-app:latest"),
		WorkspaceDomain:              env("OPL_WORKSPACE_DOMAIN", "workspace.medopl.cn"),
		WorkspaceWebUIPort:           env("OPL_WORKSPACE_WEBUI_PORT", "3000"),
		WorkspaceDataDir:             env("OPL_WORKSPACE_DATA_DIR", "/data"),
		WorkspaceProjectsDir:         env("OPL_WORKSPACE_PROJECTS_DIR", "/projects"),
		StorageClass:                 env("OPL_WORKSPACE_STORAGE_CLASS", "cbs"),
		WorkspaceVolumeSnapshotClass: os.Getenv("OPL_WORKSPACE_VOLUME_SNAPSHOT_CLASS"),
		WorkspaceNodeSelectorKey:     os.Getenv("OPL_WORKSPACE_NODE_SELECTOR_KEY"),
		WorkspaceNodeSelectorValue:   os.Getenv("OPL_WORKSPACE_NODE_SELECTOR_VALUE"),
		KubernetesNamespace:          env("OPL_K8S_NAMESPACE", "opl-cloud"),
		IngressClass:                 os.Getenv("OPL_INGRESS_CLASS"),
		ImagePullSecretName:          os.Getenv("OPL_IMAGE_PULL_SECRET_NAME"),
		OperatorToken:                os.Getenv("OPL_OPERATOR_TOKEN"),
		CodexModel:                   env("OPL_CODEX_MODEL", "gpt-5.5"),
		CodexReasoningEffort:         env("OPL_CODEX_REASONING_EFFORT", "xhigh"),
		CodexBaseURL:                 env("OPL_CODEX_BASE_URL", "https://gflabtoken.cn/v1"),
		CodexAPIKey:                  os.Getenv("OPL_CODEX_API_KEY"),
		CodexModelProvider:           env("OPL_CODEX_MODEL_PROVIDER", "gflabtoken"),
		CodexProviderName:            env("OPL_CODEX_PROVIDER_NAME", "gflabtoken"),
		CodexHome:                    env("CODEX_HOME", "/data/codex"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
```

- [ ] **Step 4: Run config tests**

Run:

```bash
cd apps/fabric-api && go test ./internal/config
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add apps/fabric-api/internal/config
git commit -m "feat: load centralized fabric config keys"
```

## Task 3: Expand Readiness To Report Central Config Gaps

**Files:**

- Modify: `apps/fabric-api/internal/service/service.go`
- Create: `apps/fabric-api/internal/service/service_test.go`
- Modify: `apps/fabric-api/cmd/fabric-api/main.go`

- [ ] **Step 1: Write service readiness tests**

Create `apps/fabric-api/internal/service/service_test.go`:

```go
package service

import (
	"slices"
	"testing"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
)

func TestReadinessRequiresProductionFabricKeys(t *testing.T) {
	svc := New(Config{Catalog: catalog.DefaultCatalog(catalog.Config{})})

	readiness := svc.Readiness()

	for _, key := range []string{
		"DATABASE_URL",
		"OPL_OPERATOR_TOKEN",
		"OPL_INGRESS_CLASS",
		"OPL_IMAGE_PULL_SECRET_NAME",
	} {
		if !slices.Contains(readiness.MissingEnv, key) {
			t.Fatalf("MissingEnv = %v, want %s", readiness.MissingEnv, key)
		}
	}
	if readiness.Ready {
		t.Fatal("readiness should be blocked with missing production keys")
	}
}

func TestReadinessAllowsOptionalCodexSecret(t *testing.T) {
	svc := New(Config{
		Catalog:             catalog.DefaultCatalog(catalog.Config{}),
		DatabaseURL:         "postgres://example",
		OperatorToken:       "operator",
		KubernetesNamespace: "opl-cloud",
		IngressClass:        "qcloud",
		ImagePullSecretName: "tcr-pull-secret",
	})

	readiness := svc.Readiness()

	if slices.Contains(readiness.MissingEnv, "OPL_CODEX_API_KEY") {
		t.Fatalf("Codex API key should be optional until workspace bootstrap is enabled: %v", readiness.MissingEnv)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd apps/fabric-api && go test ./internal/service
```

Expected: FAIL because service config does not include ingress and image pull secret fields.

- [ ] **Step 3: Extend service config and missing env checks**

Modify `apps/fabric-api/internal/service/service.go` so `Config` and `Service` include:

```go
	IngressClass        string
	ImagePullSecretName string
```

Update `New` to assign those fields.

Update `missingEnv` to require:

```go
	if s.ingressClass == "" {
		missing = append(missing, "OPL_INGRESS_CLASS")
	}
	if s.imagePullSecretName == "" {
		missing = append(missing, "OPL_IMAGE_PULL_SECRET_NAME")
	}
```

Keep `OPL_CODEX_API_KEY` optional in this task because it should block only when workspace Codex bootstrap is enabled in the provider contract.

- [ ] **Step 4: Wire config through main**

Modify `apps/fabric-api/cmd/fabric-api/main.go` service construction:

```go
	svc := service.New(service.Config{
		Catalog:             cat,
		DatabaseURL:         cfg.DatabaseURL,
		OperatorToken:       cfg.OperatorToken,
		KubernetesNamespace: cfg.KubernetesNamespace,
		IngressClass:        cfg.IngressClass,
		ImagePullSecretName: cfg.ImagePullSecretName,
	})
```

- [ ] **Step 5: Run service and HTTP tests**

Run:

```bash
cd apps/fabric-api && go test ./internal/service ./internal/http ./cmd/fabric-api
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add apps/fabric-api/internal/service apps/fabric-api/cmd/fabric-api
git commit -m "feat: report centralized config readiness gaps"
```

## Task 4: Add Workspace Runtime And Codex Config To K8s Provider Boundary

**Files:**

- Modify: `apps/fabric-api/internal/k8s/provider.go`
- Modify: `apps/fabric-api/internal/k8s/provider_test.go`

- [ ] **Step 1: Add provider test for workspace runtime env**

Append to `apps/fabric-api/internal/k8s/provider_test.go`:

```go
func TestCreateComputeInjectsWorkspaceRuntimeConfig(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{
		Client:               client,
		Namespace:            "opl-cloud",
		WorkspaceImage:       "workspace:latest",
		WorkspaceWebUIPort:   3000,
		WorkspaceDataDir:     "/data",
		WorkspaceProjectsDir: "/projects",
		CodexHome:            "/data/codex",
	}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-runtime", WorkspaceName: "Runtime", PackageID: "basic"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	deploy, err := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}

	env := map[string]string{}
	for _, item := range deploy.Spec.Template.Spec.Containers[0].Env {
		env[item.Name] = item.Value
	}
	for key, want := range map[string]string{
		"OPL_PROJECTS_DIR":     "/projects",
		"OPL_WEBUI_AUTH_MODE":  "none",
		"OPL_WORKSPACE_ROOT":   "/projects",
		"CODEX_HOME":           "/data/codex",
	} {
		if env[key] != want {
			t.Fatalf("%s = %q, want %q", key, env[key], want)
		}
	}
}
```

- [ ] **Step 2: Add provider test for Codex bootstrap secret keys**

Append:

```go
func TestCreateComputeAddsCodexSecretEnvWhenConfigured(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{
		Client:               client,
		Namespace:            "opl-cloud",
		WorkspaceImage:       "workspace:latest",
		WorkspaceWebUIPort:   3000,
		WorkspaceDataDir:     "/data",
		WorkspaceProjectsDir: "/projects",
		CodexHome:            "/data/codex",
		CodexModel:           "gpt-5.5",
		CodexReasoningEffort: "xhigh",
		CodexBaseURL:         "https://gflabtoken.cn/v1",
		CodexAPIKey:          "secret",
		CodexModelProvider:   "gflabtoken",
		CodexProviderName:    "gflabtoken",
	}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-codex", WorkspaceName: "Codex", PackageID: "basic"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	secret, err := client.CoreV1().Secrets("opl-cloud").Get(context.Background(), name+"-env", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("secret missing: %v", err)
	}

	for _, key := range []string{"OPL_CODEX_MODEL", "OPL_CODEX_REASONING_EFFORT", "OPL_CODEX_BASE_URL", "OPL_CODEX_API_KEY"} {
		if len(secret.Data[key]) == 0 {
			t.Fatalf("secret missing %s", key)
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
cd apps/fabric-api && go test ./internal/k8s
```

Expected: FAIL because provider does not create workspace env Secret or inject the new runtime env.

- [ ] **Step 4: Extend provider struct and compute manifest**

Modify `apps/fabric-api/internal/k8s/provider.go`:

```go
type Provider struct {
	Client                  kubernetes.Interface
	Namespace               string
	WorkspaceImage          string
	WorkspaceWebUIPort      int32
	WorkspaceDataDir        string
	WorkspaceProjectsDir    string
	CodexHome               string
	CodexModel              string
	CodexReasoningEffort    string
	CodexBaseURL            string
	CodexAPIKey             string
	CodexModelProvider      string
	CodexProviderName       string
}
```

Before creating the Deployment, create a Secret named `name + "-env"` when any Codex value is present:

```go
	if secret := p.codexSecret(name, labels); secret != nil {
		if _, err := p.Client.CoreV1().Secrets(p.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return CreateComputeResult{}, err
		}
	}
```

Add container env values:

```go
							{Name: "OPL_PROJECTS_DIR", Value: defaultString(p.WorkspaceProjectsDir, "/projects")},
							{Name: "OPL_WEBUI_AUTH_MODE", Value: "none"},
							{Name: "OPL_WORKSPACE_ROOT", Value: defaultString(p.WorkspaceProjectsDir, "/projects")},
							{Name: "CODEX_HOME", Value: defaultString(p.CodexHome, "/data/codex")},
```

Add helper:

```go
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
```

Add `codexSecret` helper:

```go
func (p Provider) codexSecret(name string, labels map[string]string) *corev1.Secret {
	data := map[string][]byte{}
	put := func(key, value string) {
		if value != "" {
			data[key] = []byte(value)
		}
	}
	put("OPL_CODEX_MODEL", p.CodexModel)
	put("OPL_CODEX_REASONING_EFFORT", p.CodexReasoningEffort)
	put("OPL_CODEX_BASE_URL", p.CodexBaseURL)
	put("OPL_CODEX_API_KEY", p.CodexAPIKey)
	put("OPL_CODEX_MODEL_PROVIDER", p.CodexModelProvider)
	put("OPL_CODEX_PROVIDER_NAME", p.CodexProviderName)
	if len(data) == 0 {
		return nil
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-env", Namespace: p.Namespace, Labels: labels},
		Type:       corev1.SecretTypeOpaque,
		Data:       data,
	}
}
```

- [ ] **Step 5: Run provider tests**

Run:

```bash
cd apps/fabric-api && go test ./internal/k8s
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add apps/fabric-api/internal/k8s
git commit -m "feat: carry workspace runtime config into k8s provider"
```

## Task 5: Update Kubernetes Deployment Skeleton To Reference Central Config Keys

**Files:**

- Modify: `deploy/k8s/opl-fabric-api.yaml`
- Modify: `README.md`

- [ ] **Step 1: Update deployment env**

Modify `deploy/k8s/opl-fabric-api.yaml` container env to include:

```yaml
            - name: OPL_FABRIC_CONFIG_DIR
              value: /etc/opl-fabric/config
            - name: OPL_WORKSPACE_WEBUI_PORT
              value: "3000"
            - name: OPL_WORKSPACE_DATA_DIR
              value: /data
            - name: OPL_WORKSPACE_PROJECTS_DIR
              value: /projects
            - name: OPL_INGRESS_CLASS
              value: qcloud
            - name: OPL_IMAGE_PULL_SECRET_NAME
              value: tcr-pull-secret
            - name: OPL_CODEX_MODEL
              value: gpt-5.5
            - name: OPL_CODEX_REASONING_EFFORT
              value: xhigh
            - name: OPL_CODEX_BASE_URL
              value: https://gflabtoken.cn/v1
            - name: CODEX_HOME
              value: /data/codex
            - name: OPL_CODEX_API_KEY
              valueFrom:
                secretKeyRef:
                  name: opl-fabric-api-secrets
                  key: OPL_CODEX_API_KEY
                  optional: true
```

Keep `DATABASE_URL` and `OPL_OPERATOR_TOKEN` as required secret keys.

- [ ] **Step 2: Update README config path section**

Add this section to `README.md` after `## Stack`:

```markdown
## Configuration

The default configuration catalog lives in `config/`.

Set `OPL_FABRIC_CONFIG_DIR=/path/to/config` when you need to use another path with the same file names.

Real secrets must not be committed. Use Kubernetes Secrets, external secret refs, or ignored local env files based on `config/fabric.env.example`.
```

- [ ] **Step 3: Run YAML structural check**

Run:

```bash
python3 - <<'PY'
from pathlib import Path
import yaml
docs = list(yaml.safe_load_all(Path("deploy/k8s/opl-fabric-api.yaml").read_text()))
assert any(item.get("kind") == "Deployment" for item in docs if item)
print("deployment yaml ok")
PY
```

Expected:

```text
deployment yaml ok
```

- [ ] **Step 4: Commit**

Run:

```bash
git add deploy/k8s/opl-fabric-api.yaml README.md
git commit -m "chore: document centralized fabric deployment config"
```

## Task 6: Document Remaining MedOPL-Derived Gaps

**Files:**

- Modify: `docs/status.md`
- Modify: `docs/decisions.md`

- [ ] **Step 1: Update status**

Add to `docs/status.md` under `Remaining risks`:

```markdown
- Central config now records medopl-3 TKE and Codex workspace inputs, but provider behavior still needs full PVC, Secret, Ingress, VolumeSnapshot, backup, restore, reconcile, and status implementation through Go client-go.
- `OPL_CODEX_API_KEY` is optional until workspace Codex bootstrap is enabled for a published mutating compute API.
```

- [ ] **Step 2: Update decisions**

Add to `docs/decisions.md`:

```markdown
## 2026-07-04: Central Fabric config directory

OPL Fabric uses `config/` as the default configuration root. Operators can move the directory and set `OPL_FABRIC_CONFIG_DIR` to the new path.

The initial config catalog imports public deployment and provider key names from `/home/dev/medopl-3` at commit `d2c7474deb6deb39daf81232f563a5f39c0fdd16`. Real secrets are not imported; only key names, defaults, workspace runtime paths, readiness checks, and secretRef shapes are retained.
```

- [ ] **Step 3: Commit**

Run:

```bash
git add docs/status.md docs/decisions.md
git commit -m "docs: record centralized config decision"
```

## Task 7: Full Verification And Push

**Files:**

- Verify all changed files.

- [ ] **Step 1: Run full tests**

Run:

```bash
npm test
```

Expected:

```text
contracts pass
go test ./... pass
console source tests pass
console typecheck pass
console build pass
```

- [ ] **Step 2: Run diff whitespace check**

Run:

```bash
git diff --check HEAD
```

Expected: no output.

- [ ] **Step 3: Confirm branch state**

Run:

```bash
git status --short --branch
```

Expected: branch is clean and ahead of `origin/main` by the new commits.

- [ ] **Step 4: Push**

Run:

```bash
git push origin main
```

Expected: GitHub `main` points to the final commit.

## Self-Review

Spec coverage:

- Central folder: Task 1.
- Movable path: Task 1 and Task 2 through `OPL_FABRIC_CONFIG_DIR`.
- medopl-3-derived config: Task 1 source baseline and config files.
- Required language stack: Go changes in Tasks 2-4, React unchanged, PostgreSQL unchanged, K8s remains Go `client-go`.
- Secret handling: Task 1 and Task 5 keep real values out of git.
- Deployment docs: Task 5.
- Remaining risk documentation: Task 6.
- Verification and push: Task 7.

Placeholder scan:

- No `TBD` markers.
- No implementation steps without file paths.
- Optional future provider work is explicitly documented as remaining risk, not hidden inside vague steps.

Type consistency:

- `ConfigDir`, workspace runtime fields, TKE fields, and Codex fields are introduced in `internal/config` before service/provider usage.
- `OPL_FABRIC_CONFIG_DIR` is consistently the override key.
- `config/` is consistently the default directory.
