package staging

import "testing"

func TestGateBlocksWhenRequiredCloudInputsAreMissing(t *testing.T) {
	result := EvaluateGate(Config{
		DatabaseURL: "postgres://example",
		Namespace:   "opl-fabric",
	})

	if result.Ready {
		t.Fatal("gate should block with missing cloud inputs")
	}
	for _, key := range []string{
		"TENCENT_DEPLOY_KUBECONFIG_REF",
		"TENCENT_DEPLOY_CLUSTER_ID",
		"TENCENT_TKE_REGION",
		"TENCENT_MUTATION_SECRET_ID",
		"TENCENT_MUTATION_SECRET_KEY",
		"TENCENT_TCR_REGISTRY",
		"TENCENT_TCR_NAMESPACE",
		"TENCENT_TCR_REGION",
		"OPL_WORKSPACE_STORAGE_CLASS",
		"OPL_INGRESS_CLASS",
		"OPL_WORKSPACE_DOMAIN",
		"OPL_IMAGE_PULL_SECRET_NAME",
	} {
		if !contains(result.Missing, key) {
			t.Fatalf("missing = %v, want %s", result.Missing, key)
		}
	}
}

func TestGateRequiresExplicitLiveMutationAllowance(t *testing.T) {
	cfg := completeConfig()
	cfg.AllowNodePoolMutation = true
	cfg.AllowStagingE2E = false

	result := EvaluateGate(cfg)

	if result.Ready {
		t.Fatal("gate should block without staging e2e allowance")
	}
	if !contains(result.Blockers, "staging_e2e_not_allowed") {
		t.Fatalf("blockers = %v", result.Blockers)
	}
}

func TestGateAllowsDryRunWithoutLiveMutation(t *testing.T) {
	cfg := completeConfig()
	cfg.AllowNodePoolMutation = false
	cfg.AllowStagingE2E = false

	result := EvaluateGate(cfg)

	if !result.Ready {
		t.Fatalf("dry gate should be ready: %+v", result)
	}
	if result.Mode != "dry_run" {
		t.Fatalf("mode = %q, want dry_run", result.Mode)
	}
}

func TestGateAllowsLiveOnlyWhenBothFlagsAreExplicit(t *testing.T) {
	cfg := completeConfig()
	cfg.AllowNodePoolMutation = true
	cfg.AllowStagingE2E = true

	result := EvaluateGate(cfg)

	if !result.Ready {
		t.Fatalf("live gate should be ready: %+v", result)
	}
	if result.Mode != "live_staging" {
		t.Fatalf("mode = %q, want live_staging", result.Mode)
	}
}

func completeConfig() Config {
	return Config{
		DatabaseURL:             "postgres://example",
		OperatorToken:           "token",
		KubeconfigRef:           "secret/kubeconfig",
		Namespace:               "opl-fabric",
		StorageClass:            "cbs",
		IngressClass:            "qcloud",
		WorkspaceDomain:         "workspace.medopl.cn",
		ImagePullSecretName:     "tcr-pull-secret",
		TencentClusterID:        "cls-example",
		TencentRegion:           "ap-guangzhou",
		TencentSecretID:         "secret-id",
		TencentSecretKey:        "secret-key",
		TencentTCRRegistry:      "registry.example.com",
		TencentTCRNamespace:     "opl",
		TencentTCRRegion:        "ap-guangzhou",
		NodePoolLaunchJSON:      `{"InstanceType":"SA5.LARGE8"}`,
		NodePoolAutoscalingJSON: `{"MinSize":0,"MaxSize":3}`,
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
