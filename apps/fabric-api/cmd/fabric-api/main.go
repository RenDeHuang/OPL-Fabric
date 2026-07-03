package main

import (
	"log"
	"net/http"
	"time"

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
	handler := httpapi.NewServer(svc, httpapi.Config{OperatorToken: cfg.OperatorToken})

	addr := ":" + cfg.Port
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("fabric API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
