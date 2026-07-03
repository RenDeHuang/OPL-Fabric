package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
)

type Config struct {
	OperatorToken string
}

func NewServer(svc *service.Service, cfg Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/fabric/readiness", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, svc.Readiness())
	})
	mux.HandleFunc("GET /api/fabric/catalog", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, svc.Catalog())
	})

	return requireOperatorToken(cfg.OperatorToken, mux)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

func requireOperatorToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authorized(token, r.Header.Get("Authorization")) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="opl-fabric"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authorized(token, header string) bool {
	if token == "" {
		return false
	}
	got, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}
