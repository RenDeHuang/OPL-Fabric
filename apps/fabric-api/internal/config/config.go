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
