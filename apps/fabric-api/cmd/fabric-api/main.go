package main

import (
	"log"
	"net/http"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/config"
	httpapi "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/http"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
)

func main() {
	cfg := config.Load()
	cat := catalog.DefaultCatalog(catalog.Config{
		WorkspaceImage:  cfg.WorkspaceImage,
		WorkspaceDomain: cfg.WorkspaceDomain,
		StorageClass:    cfg.StorageClass,
	})
	svc := service.New(service.Config{Catalog: cat})
	server := httpapi.NewServer(svc)

	addr := ":" + cfg.Port
	log.Printf("fabric API listening on %s", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatal(err)
	}
}
