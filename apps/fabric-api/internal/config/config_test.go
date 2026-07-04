package config

import "testing"

func TestLoadReadsFabricConfigDirAndWorkspaceDefaults(t *testing.T) {
	t.Setenv("OPL_FABRIC_CONFIG_DIR", "/tmp/opl-fabric-config")
	t.Setenv("OPL_WORKSPACE_WEBUI_PORT", "3001")
	t.Setenv("OPL_WORKSPACE_DATA_DIR", "/workspace-data")
	t.Setenv("OPL_WORKSPACE_PROJECTS_DIR", "/workspace-projects")
	t.Setenv("OPL_WORKSPACE_NODE_SELECTOR_KEY", "medopl.cn/workload")
	t.Setenv("OPL_WORKSPACE_NODE_SELECTOR_VALUE", "medopl")
	t.Setenv("OPL_INGRESS_CLASS", "qcloud")
	t.Setenv("OPL_IMAGE_PULL_SECRET_NAME", "tcr-pull-secret")
	t.Setenv("TENCENT_TKE_REGION", "ap-guangzhou")
	t.Setenv("TENCENT_DEPLOY_KUBECONFIG_REF", "opl-fabric/tencent-deploy-kubeconfig")
	t.Setenv("TENCENT_DEPLOY_CLUSTER_ID", "cls-example")
	t.Setenv("TENCENT_MUTATION_SECRET_ID", "secret-id")
	t.Setenv("TENCENT_MUTATION_SECRET_KEY", "secret-key")
	t.Setenv("TENCENT_CVM_SUBNET_ID", "subnet-1")
	t.Setenv("TENCENT_CVM_SECURITY_GROUP_IDS", "sg-1")
	t.Setenv("TENCENT_CVM_SYSTEM_DISK_TYPE", "CLOUD_BSSD")
	t.Setenv("TENCENT_CVM_SYSTEM_DISK_SIZE_GB", "50")
	t.Setenv("OPL_TKE_INSTANCE_CHARGE_TYPE", "POSTPAID_BY_HOUR")
	t.Setenv("OPL_STAGING_E2E_ALLOW_LIVE", "true")
	t.Setenv("OPL_CODEX_MODEL", "gpt-5.5")
	t.Setenv("OPL_CODEX_REASONING_EFFORT", "xhigh")
	t.Setenv("OPL_CODEX_BASE_URL", "https://gflabtoken.cn/v1")
	t.Setenv("OPL_CODEX_API_KEY", "secret")
	t.Setenv("OPL_CODEX_MODEL_PROVIDER", "gflabtoken")
	t.Setenv("OPL_CODEX_PROVIDER_NAME", "gflabtoken")
	t.Setenv("CODEX_HOME", "/data/codex")
	t.Setenv("OPL_FABRIC_WORKER_ENABLED", "true")
	t.Setenv("OPL_FABRIC_WORKER_OWNER", "fabric-api-1")
	t.Setenv("OPL_FABRIC_WORKER_INTERVAL", "2s")
	t.Setenv("OPL_FABRIC_WORKER_LEASE_TTL", "30s")
	t.Setenv("OPL_FABRIC_WORKER_BATCH_SIZE", "5")

	cfg := Load()

	if cfg.ConfigDir != "/tmp/opl-fabric-config" {
		t.Fatalf("ConfigDir = %q", cfg.ConfigDir)
	}
	if cfg.WorkspaceWebUIPort != "3001" {
		t.Fatalf("WorkspaceWebUIPort = %q", cfg.WorkspaceWebUIPort)
	}
	if cfg.TencentTKERegion != "ap-guangzhou" {
		t.Fatalf("TencentTKERegion = %q", cfg.TencentTKERegion)
	}
	if cfg.TencentMutationSecretID != "secret-id" {
		t.Fatalf("TencentMutationSecretID not loaded")
	}
	if cfg.TencentCVMSubnetIDs != "subnet-1" || cfg.TencentCVMSecurityGroupIDs != "sg-1" {
		t.Fatalf("Tencent CVM network config not loaded: %+v", cfg)
	}
	if cfg.TencentCVMSystemDiskType != "CLOUD_BSSD" || cfg.TencentCVMSystemDiskSizeGB != "50" {
		t.Fatalf("Tencent CVM disk config not loaded: %+v", cfg)
	}
	if cfg.StagingE2EAllowLive != "true" {
		t.Fatalf("StagingE2EAllowLive = %q", cfg.StagingE2EAllowLive)
	}
	if cfg.CodexAPIKey != "secret" {
		t.Fatalf("CodexAPIKey not loaded")
	}
	if cfg.WorkerEnabled != "true" || cfg.WorkerOwner != "fabric-api-1" || cfg.WorkerInterval != "2s" || cfg.WorkerLeaseTTL != "30s" || cfg.WorkerBatchSize != "5" {
		t.Fatalf("worker config not loaded: %+v", cfg)
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
	if cfg.KubernetesNamespace != "opl-fabric" {
		t.Fatalf("KubernetesNamespace = %q", cfg.KubernetesNamespace)
	}
	if cfg.TKEInstanceChargeType != "POSTPAID_BY_HOUR" {
		t.Fatalf("TKEInstanceChargeType = %q", cfg.TKEInstanceChargeType)
	}
	if cfg.TKEAllowNodePoolMutation != "false" {
		t.Fatalf("TKEAllowNodePoolMutation = %q", cfg.TKEAllowNodePoolMutation)
	}
	if cfg.StagingE2EAllowLive != "false" {
		t.Fatalf("StagingE2EAllowLive = %q", cfg.StagingE2EAllowLive)
	}
	if cfg.WorkerEnabled != "false" {
		t.Fatalf("WorkerEnabled = %q", cfg.WorkerEnabled)
	}
	if cfg.WorkerOwner != "fabric-api" {
		t.Fatalf("WorkerOwner = %q", cfg.WorkerOwner)
	}
	if cfg.WorkerInterval != "5s" || cfg.WorkerLeaseTTL != "60s" || cfg.WorkerBatchSize != "10" {
		t.Fatalf("worker defaults = interval:%q lease:%q batch:%q", cfg.WorkerInterval, cfg.WorkerLeaseTTL, cfg.WorkerBatchSize)
	}
}
