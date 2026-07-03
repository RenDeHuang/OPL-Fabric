package service

import "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"

type Config struct {
	Catalog catalog.Catalog
}

type Service struct {
	catalog catalog.Catalog
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
	return &Service{catalog: cfg.Catalog}
}

func (s *Service) Catalog() catalog.Catalog {
	return s.catalog
}

func (s *Service) Readiness() Readiness {
	return Readiness{
		Ready:           true,
		Provider:        "tencent-tke",
		MissingEnv:      []string{},
		ResourceCatalog: s.catalog,
		Blockers:        []string{},
		RepairHints:     []string{},
	}
}
