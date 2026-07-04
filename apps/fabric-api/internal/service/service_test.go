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
		"TENCENT_TKE_REGION",
		"TENCENT_MUTATION_SECRET_ID",
		"TENCENT_MUTATION_SECRET_KEY",
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
		Catalog:                    catalog.DefaultCatalog(catalog.Config{}),
		DatabaseURL:                "postgres://example",
		OperatorToken:              "operator",
		KubernetesNamespace:        "opl-fabric",
		IngressClass:               "qcloud",
		ImagePullSecretName:        "tcr-pull-secret",
		WorkspaceImage:             "workspace:latest",
		WorkspaceDomain:            "workspace.medopl.cn",
		StorageClass:               "cbs",
		TencentTKERegion:           "ap-guangzhou",
		TencentClusterID:           "cls-example",
		TencentSecretID:            "secret-id",
		TencentSecretKey:           "secret-key",
		TencentTCRRegistry:         "registry.example.com",
		TencentTCRNamespace:        "opl",
		TencentTCRRegion:           "ap-guangzhou",
		TencentCVMSubnetIDs:        "subnet-1",
		TencentCVMSecurityGroupIDs: "sg-1",
	})

	readiness := svc.Readiness()

	if slices.Contains(readiness.MissingEnv, "OPL_CODEX_API_KEY") {
		t.Fatalf("Codex API key should be optional until workspace bootstrap is enabled: %v", readiness.MissingEnv)
	}
}
