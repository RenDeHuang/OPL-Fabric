package staging

import "testing"

func TestGateBlocksWhenRequiredCloudInputsAreMissing(t *testing.T) {
	result := EvaluateGate(Config{
		DatabaseURL: "postgres://example",
		Namespace:   "oplfabric",
	})

	if result.Ready {
		t.Fatal("gate should block with missing cloud inputs")
	}
	if result.Mode != "blocked" {
		t.Fatalf("mode = %q, want blocked", result.Mode)
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
		"TENCENT_CVM_SUBNET_ID",
		"TENCENT_CVM_SECURITY_GROUP_IDS",
		"OPL_WORKSPACE_STORAGE_CLASS",
		"OPL_INGRESS_CLASS",
		"OPL_WORKSPACE_DOMAIN",
		"OPL_WORKSPACE_IMAGE",
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
	cfg.WorkerEnabled = true

	result := EvaluateGate(cfg)

	if result.Ready {
		t.Fatal("gate should block without staging e2e allowance")
	}
	if !contains(result.Blockers, "staging_e2e_not_allowed") {
		t.Fatalf("blockers = %v", result.Blockers)
	}
	if result.Mode != "blocked" {
		t.Fatalf("mode = %q, want blocked", result.Mode)
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
	cfg.WorkerEnabled = true

	result := EvaluateGate(cfg)

	if !result.Ready {
		t.Fatalf("live gate should be ready: %+v", result)
	}
	if result.Mode != "ready_for_live" {
		t.Fatalf("mode = %q, want ready_for_live", result.Mode)
	}
}

func TestGateBlocksLiveWhenWorkerIsNotEnabled(t *testing.T) {
	cfg := completeConfig()
	cfg.AllowNodePoolMutation = true
	cfg.AllowStagingE2E = true
	cfg.WorkerEnabled = false

	result := EvaluateGate(cfg)

	if result.Ready {
		t.Fatal("gate should block live e2e without worker")
	}
	if !contains(result.Blockers, "fabric_worker_not_enabled") {
		t.Fatalf("blockers = %v", result.Blockers)
	}
	if result.Mode != "blocked" {
		t.Fatalf("mode = %q, want blocked", result.Mode)
	}
}

func completeConfig() Config {
	return Config{
		DatabaseURL:                "postgres://example",
		OperatorToken:              "token",
		KubeconfigRef:              "secret/kubeconfig",
		Namespace:                  "oplfabric",
		StorageClass:               "cbs",
		IngressClass:               "qcloud",
		WorkspaceDomain:            "workspace.medopl.cn",
		WorkspaceImage:             "tcr.example.com/opl/workspace:staging",
		ImagePullSecretName:        "tcr-pull-secret",
		TencentClusterID:           "cls-example",
		TencentRegion:              "ap-guangzhou",
		TencentSecretID:            "secret-id",
		TencentSecretKey:           "secret-key",
		TencentTCRRegistry:         "registry.example.com",
		TencentTCRNamespace:        "opl",
		TencentTCRRegion:           "ap-guangzhou",
		TencentCVMSubnetIDs:        "subnet-1",
		TencentCVMSecurityGroupIDs: "sg-1",
		TencentCVMSystemDiskType:   "CLOUD_BSSD",
		TencentCVMSystemDiskSizeGB: "50",
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
