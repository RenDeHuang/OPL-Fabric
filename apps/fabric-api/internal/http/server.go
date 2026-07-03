package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
)

func NewServer(svc *service.Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/fabric/readiness", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, svc.Readiness())
	})
	mux.HandleFunc("GET /api/fabric/catalog", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, svc.Catalog())
	})

	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}
