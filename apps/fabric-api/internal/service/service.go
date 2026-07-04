package service

import "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"

type Config struct {
	Catalog             catalog.Catalog
	DatabaseURL         string
	OperatorToken       string
	KubernetesNamespace string
	IngressClass        string
	ImagePullSecretName string
	WorkspaceImage      string
	WorkspaceDomain     string
	StorageClass        string
	TencentTKERegion    string
	TencentClusterID    string
	TencentSecretID     string
	TencentSecretKey    string
	TencentTCRRegistry  string
	TencentTCRNamespace string
	TencentTCRRegion    string
}

type Service struct {
	catalog             catalog.Catalog
	databaseURL         string
	operatorToken       string
	kubernetesNamespace string
	ingressClass        string
	imagePullSecretName string
	workspaceImage      string
	workspaceDomain     string
	storageClass        string
	tencentTKERegion    string
	tencentClusterID    string
	tencentSecretID     string
	tencentSecretKey    string
	tencentTCRRegistry  string
	tencentTCRNamespace string
	tencentTCRRegion    string
}

type Readiness struct {
	Ready           bool            `json:"ready"`
	Provider        string          `json:"provider"`
	MissingEnv      []string        `json:"missingEnv"`
	ResourceCatalog catalog.Catalog `json:"resourceCatalog"`
	Blockers        []string        `json:"blockers"`
	RepairHints     []string        `json:"repairHints"`
}

func New(cfg Config) *Service {
	return &Service{
		catalog:             cfg.Catalog,
		databaseURL:         cfg.DatabaseURL,
		operatorToken:       cfg.OperatorToken,
		kubernetesNamespace: cfg.KubernetesNamespace,
		ingressClass:        cfg.IngressClass,
		imagePullSecretName: cfg.ImagePullSecretName,
		workspaceImage:      cfg.WorkspaceImage,
		workspaceDomain:     cfg.WorkspaceDomain,
		storageClass:        cfg.StorageClass,
		tencentTKERegion:    cfg.TencentTKERegion,
		tencentClusterID:    cfg.TencentClusterID,
		tencentSecretID:     cfg.TencentSecretID,
		tencentSecretKey:    cfg.TencentSecretKey,
		tencentTCRRegistry:  cfg.TencentTCRRegistry,
		tencentTCRNamespace: cfg.TencentTCRNamespace,
		tencentTCRRegion:    cfg.TencentTCRRegion,
	}
}

func (s *Service) Catalog() catalog.Catalog {
	return s.catalog
}

func (s *Service) Readiness() Readiness {
	missingEnv := s.missingEnv()
	return Readiness{
		Ready:           len(missingEnv) == 0,
		Provider:        "tencent-tke",
		MissingEnv:      missingEnv,
		ResourceCatalog: s.catalog,
		Blockers:        blockersForMissingEnv(missingEnv),
		RepairHints:     repairHintsForMissingEnv(missingEnv),
	}
}

func (s *Service) missingEnv() []string {
	missing := []string{}
	if s.databaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if s.operatorToken == "" {
		missing = append(missing, "OPL_OPERATOR_TOKEN")
	}
	if s.kubernetesNamespace == "" {
		missing = append(missing, "OPL_K8S_NAMESPACE")
	}
	if s.ingressClass == "" {
		missing = append(missing, "OPL_INGRESS_CLASS")
	}
	if s.imagePullSecretName == "" {
		missing = append(missing, "OPL_IMAGE_PULL_SECRET_NAME")
	}
	if s.workspaceImage == "" {
		missing = append(missing, "OPL_WORKSPACE_IMAGE")
	}
	if s.workspaceDomain == "" {
		missing = append(missing, "OPL_WORKSPACE_DOMAIN")
	}
	if s.storageClass == "" {
		missing = append(missing, "OPL_WORKSPACE_STORAGE_CLASS")
	}
	if s.tencentTKERegion == "" {
		missing = append(missing, "TENCENT_TKE_REGION")
	}
	if s.tencentClusterID == "" {
		missing = append(missing, "TENCENT_DEPLOY_CLUSTER_ID")
	}
	if s.tencentSecretID == "" {
		missing = append(missing, "TENCENT_MUTATION_SECRET_ID")
	}
	if s.tencentSecretKey == "" {
		missing = append(missing, "TENCENT_MUTATION_SECRET_KEY")
	}
	if s.tencentTCRRegistry == "" {
		missing = append(missing, "TENCENT_TCR_REGISTRY")
	}
	if s.tencentTCRNamespace == "" {
		missing = append(missing, "TENCENT_TCR_NAMESPACE")
	}
	if s.tencentTCRRegion == "" {
		missing = append(missing, "TENCENT_TCR_REGION")
	}
	return missing
}

func blockersForMissingEnv(missingEnv []string) []string {
	blockers := make([]string, 0, len(missingEnv))
	for _, name := range missingEnv {
		blockers = append(blockers, "missing_env:"+name)
	}
	return blockers
}

func repairHintsForMissingEnv(missingEnv []string) []string {
	hints := make([]string, 0, len(missingEnv))
	for _, name := range missingEnv {
		hints = append(hints, "set "+name+" before serving Fabric traffic")
	}
	return hints
}
