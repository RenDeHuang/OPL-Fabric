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
	t.Setenv("OPL_TKE_NODEPOOL_AUTOSCALING_GROUP_PARA_JSON", `{"MinSize":0,"MaxSize":3}`)
	t.Setenv("OPL_TKE_NODEPOOL_LAUNCH_CONFIGURE_PARA_JSON", `{"SystemDisk":{"DiskType":"CLOUD_BSSD"}}`)
	t.Setenv("OPL_TKE_INSTANCE_CHARGE_TYPE", "POSTPAID_BY_HOUR")
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
	if cfg.TencentTKERegion != "ap-guangzhou" {
		t.Fatalf("TencentTKERegion = %q", cfg.TencentTKERegion)
	}
	if cfg.TencentMutationSecretID != "secret-id" {
		t.Fatalf("TencentMutationSecretID not loaded")
	}
	if cfg.TKENodePoolAutoscalingJSON == "" || cfg.TKENodePoolLaunchJSON == "" {
		t.Fatalf("TKE node pool JSON inputs not loaded")
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
	if cfg.KubernetesNamespace != "opl-fabric" {
		t.Fatalf("KubernetesNamespace = %q", cfg.KubernetesNamespace)
	}
	if cfg.TKEInstanceChargeType != "POSTPAID_BY_HOUR" {
		t.Fatalf("TKEInstanceChargeType = %q", cfg.TKEInstanceChargeType)
	}
	if cfg.TKEAllowNodePoolMutation != "true" {
		t.Fatalf("TKEAllowNodePoolMutation = %q", cfg.TKEAllowNodePoolMutation)
	}
}
