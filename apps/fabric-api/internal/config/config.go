package config

import "os"

type Config struct {
	Port                string
	DatabaseURL         string
	WorkspaceImage      string
	WorkspaceDomain     string
	StorageClass        string
	KubernetesNamespace string
	OperatorToken       string
}

func Load() Config {
	return Config{
		Port:                env("PORT", "8787"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		WorkspaceImage:      env("OPL_WORKSPACE_IMAGE", "ghcr.io/gaofeng21cn/one-person-lab-app:latest"),
		WorkspaceDomain:     env("OPL_WORKSPACE_DOMAIN", "workspace.medopl.cn"),
		StorageClass:        env("OPL_WORKSPACE_STORAGE_CLASS", "cbs"),
		KubernetesNamespace: env("OPL_K8S_NAMESPACE", "opl-cloud"),
		OperatorToken:       os.Getenv("OPL_OPERATOR_TOKEN"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
