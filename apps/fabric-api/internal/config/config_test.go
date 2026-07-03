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
