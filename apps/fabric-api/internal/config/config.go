package config

import "os"

type Config struct {
	Port                        string
	ConfigDir                   string
	DatabaseURL                 string
	WorkspaceImage              string
	WorkspaceDomain             string
	WorkspaceWebUIPort          string
	WorkspaceDataDir            string
	WorkspaceProjectsDir        string
	StorageClass                string
	WorkspaceNodeSelectorKey    string
	WorkspaceNodeSelectorValue  string
	KubernetesNamespace         string
	IngressClass                string
	ImagePullSecretName         string
	TencentTKERegion            string
	TencentDeployKubeconfigRef  string
	TencentDeployClusterID      string
	TencentMutationSecretID     string
	TencentMutationSecretKey    string
	TencentTCRRegistry          string
	TencentTCRNamespace         string
	TencentTCRRegion            string
	TKENodePoolAutoscalingJSON  string
	TKENodePoolLaunchJSON       string
	TKEInstanceChargeType       string
	TKENodePoolDesiredPodNumber string
	TKEAllowNodePoolMutation    string
	StagingE2EAllowLive         string
	OperatorToken               string
	CodexModel                  string
	CodexReasoningEffort        string
	CodexBaseURL                string
	CodexAPIKey                 string
	CodexModelProvider          string
	CodexProviderName           string
	CodexHome                   string
	WorkerEnabled               string
	WorkerOwner                 string
	WorkerInterval              string
	WorkerLeaseTTL              string
	WorkerBatchSize             string
}

func Load() Config {
	return Config{
		Port:                        env("PORT", "8787"),
		ConfigDir:                   env("OPL_FABRIC_CONFIG_DIR", "config"),
		DatabaseURL:                 os.Getenv("DATABASE_URL"),
		WorkspaceImage:              env("OPL_WORKSPACE_IMAGE", "ghcr.io/gaofeng21cn/one-person-lab-app:latest"),
		WorkspaceDomain:             env("OPL_WORKSPACE_DOMAIN", "workspace.medopl.cn"),
		WorkspaceWebUIPort:          env("OPL_WORKSPACE_WEBUI_PORT", "3000"),
		WorkspaceDataDir:            env("OPL_WORKSPACE_DATA_DIR", "/data"),
		WorkspaceProjectsDir:        env("OPL_WORKSPACE_PROJECTS_DIR", "/projects"),
		StorageClass:                env("OPL_WORKSPACE_STORAGE_CLASS", "cbs"),
		WorkspaceNodeSelectorKey:    os.Getenv("OPL_WORKSPACE_NODE_SELECTOR_KEY"),
		WorkspaceNodeSelectorValue:  os.Getenv("OPL_WORKSPACE_NODE_SELECTOR_VALUE"),
		KubernetesNamespace:         env("OPL_K8S_NAMESPACE", "opl-fabric"),
		IngressClass:                os.Getenv("OPL_INGRESS_CLASS"),
		ImagePullSecretName:         os.Getenv("OPL_IMAGE_PULL_SECRET_NAME"),
		TencentTKERegion:            os.Getenv("TENCENT_TKE_REGION"),
		TencentDeployKubeconfigRef:  os.Getenv("TENCENT_DEPLOY_KUBECONFIG_REF"),
		TencentDeployClusterID:      os.Getenv("TENCENT_DEPLOY_CLUSTER_ID"),
		TencentMutationSecretID:     os.Getenv("TENCENT_MUTATION_SECRET_ID"),
		TencentMutationSecretKey:    os.Getenv("TENCENT_MUTATION_SECRET_KEY"),
		TencentTCRRegistry:          os.Getenv("TENCENT_TCR_REGISTRY"),
		TencentTCRNamespace:         os.Getenv("TENCENT_TCR_NAMESPACE"),
		TencentTCRRegion:            os.Getenv("TENCENT_TCR_REGION"),
		TKENodePoolAutoscalingJSON:  os.Getenv("OPL_TKE_NODEPOOL_AUTOSCALING_GROUP_PARA_JSON"),
		TKENodePoolLaunchJSON:       os.Getenv("OPL_TKE_NODEPOOL_LAUNCH_CONFIGURE_PARA_JSON"),
		TKEInstanceChargeType:       env("OPL_TKE_INSTANCE_CHARGE_TYPE", "POSTPAID_BY_HOUR"),
		TKENodePoolDesiredPodNumber: os.Getenv("OPL_TKE_NODEPOOL_DESIRED_POD_NUMBER"),
		TKEAllowNodePoolMutation:    env("OPL_TKE_ALLOW_NODEPOOL_MUTATION", "false"),
		StagingE2EAllowLive:         env("OPL_STAGING_E2E_ALLOW_LIVE", "false"),
		OperatorToken:               os.Getenv("OPL_OPERATOR_TOKEN"),
		CodexModel:                  env("OPL_CODEX_MODEL", "gpt-5.5"),
		CodexReasoningEffort:        env("OPL_CODEX_REASONING_EFFORT", "xhigh"),
		CodexBaseURL:                env("OPL_CODEX_BASE_URL", "https://gflabtoken.cn/v1"),
		CodexAPIKey:                 os.Getenv("OPL_CODEX_API_KEY"),
		CodexModelProvider:          env("OPL_CODEX_MODEL_PROVIDER", "gflabtoken"),
		CodexProviderName:           env("OPL_CODEX_PROVIDER_NAME", "gflabtoken"),
		CodexHome:                   env("CODEX_HOME", "/data/codex"),
		WorkerEnabled:               env("OPL_FABRIC_WORKER_ENABLED", "false"),
		WorkerOwner:                 env("OPL_FABRIC_WORKER_OWNER", "fabric-api"),
		WorkerInterval:              env("OPL_FABRIC_WORKER_INTERVAL", "5s"),
		WorkerLeaseTTL:              env("OPL_FABRIC_WORKER_LEASE_TTL", "60s"),
		WorkerBatchSize:             env("OPL_FABRIC_WORKER_BATCH_SIZE", "10"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
