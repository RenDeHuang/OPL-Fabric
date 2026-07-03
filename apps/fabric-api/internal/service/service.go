package service

import "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"

type Config struct {
	Catalog             catalog.Catalog
	DatabaseURL         string
	OperatorToken       string
	KubernetesNamespace string
	IngressClass        string
	ImagePullSecretName string
}

type Service struct {
	catalog             catalog.Catalog
	databaseURL         string
	operatorToken       string
	kubernetesNamespace string
	ingressClass        string
	imagePullSecretName string
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
