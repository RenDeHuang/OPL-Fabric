package staging

type Config struct {
	DatabaseURL             string
	OperatorToken           string
	KubeconfigRef           string
	Namespace               string
	StorageClass            string
	IngressClass            string
	WorkspaceDomain         string
	WorkspaceImage          string
	ImagePullSecretName     string
	TencentClusterID        string
	TencentRegion           string
	TencentSecretID         string
	TencentSecretKey        string
	TencentTCRRegistry      string
	TencentTCRNamespace     string
	TencentTCRRegion        string
	NodePoolLaunchJSON      string
	NodePoolAutoscalingJSON string
	AllowNodePoolMutation   bool
	AllowStagingE2E         bool
	WorkerEnabled           bool
}

type Result struct {
	Ready    bool
	Mode     string
	Missing  []string
	Blockers []string
}

func EvaluateGate(cfg Config) Result {
	missing := []string{}
	require := func(key, value string) {
		if value == "" {
			missing = append(missing, key)
		}
	}
	require("DATABASE_URL", cfg.DatabaseURL)
	require("OPL_OPERATOR_TOKEN", cfg.OperatorToken)
	require("TENCENT_DEPLOY_KUBECONFIG_REF", cfg.KubeconfigRef)
	require("OPL_K8S_NAMESPACE", cfg.Namespace)
	require("OPL_WORKSPACE_STORAGE_CLASS", cfg.StorageClass)
	require("OPL_INGRESS_CLASS", cfg.IngressClass)
	require("OPL_WORKSPACE_DOMAIN", cfg.WorkspaceDomain)
	require("OPL_WORKSPACE_IMAGE", cfg.WorkspaceImage)
	require("OPL_IMAGE_PULL_SECRET_NAME", cfg.ImagePullSecretName)
	require("TENCENT_DEPLOY_CLUSTER_ID", cfg.TencentClusterID)
	require("TENCENT_TKE_REGION", cfg.TencentRegion)
	require("TENCENT_MUTATION_SECRET_ID", cfg.TencentSecretID)
	require("TENCENT_MUTATION_SECRET_KEY", cfg.TencentSecretKey)
	require("TENCENT_TCR_REGISTRY", cfg.TencentTCRRegistry)
	require("TENCENT_TCR_NAMESPACE", cfg.TencentTCRNamespace)
	require("TENCENT_TCR_REGION", cfg.TencentTCRRegion)
	require("OPL_TKE_NODEPOOL_LAUNCH_CONFIGURE_PARA_JSON", cfg.NodePoolLaunchJSON)
	require("OPL_TKE_NODEPOOL_AUTOSCALING_GROUP_PARA_JSON", cfg.NodePoolAutoscalingJSON)

	blockers := []string{}
	mode := "dry_run"
	if cfg.AllowNodePoolMutation {
		if !cfg.AllowStagingE2E {
			blockers = append(blockers, "staging_e2e_not_allowed")
		}
		if !cfg.WorkerEnabled {
			blockers = append(blockers, "fabric_worker_not_enabled")
		}
		if len(blockers) == 0 {
			mode = "ready_for_live"
		}
	}
	if len(missing) > 0 || len(blockers) > 0 {
		mode = "blocked"
	}
	return Result{
		Ready:    len(missing) == 0 && len(blockers) == 0,
		Mode:     mode,
		Missing:  missing,
		Blockers: blockers,
	}
}
