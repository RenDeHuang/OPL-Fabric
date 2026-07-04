package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
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
	mux.HandleFunc("POST /api/fabric/storage-volumes", func(w http.ResponseWriter, r *http.Request) {
		handleMutation(w, r, svc.AcceptStorageVolume)
	})
	mux.HandleFunc("POST /api/fabric/compute-resources", func(w http.ResponseWriter, r *http.Request) {
		handleMutation(w, r, svc.AcceptComputeResource)
	})
	mux.HandleFunc("POST /api/fabric/storage-attachments", func(w http.ResponseWriter, r *http.Request) {
		handleMutation(w, r, svc.AcceptStorageAttachment)
	})
	mux.HandleFunc("POST /api/fabric/workspace-entries", func(w http.ResponseWriter, r *http.Request) {
		handleMutation(w, r, svc.AcceptWorkspaceEntry)
	})
	mux.HandleFunc("POST /api/fabric/compute-resources/{id}/destroy", func(w http.ResponseWriter, r *http.Request) {
		handleConfirmedMutation(w, r, func(headers service.MutationHeaders, req service.ConfirmRequest) (service.OperationReceipt, error) {
			return svc.AcceptComputeDestroy(r.Context(), headers, r.PathValue("id"), req)
		})
	})
	mux.HandleFunc("POST /api/fabric/storage-volumes/{id}/destroy", func(w http.ResponseWriter, r *http.Request) {
		handleConfirmedMutation(w, r, func(headers service.MutationHeaders, req service.ConfirmRequest) (service.OperationReceipt, error) {
			return svc.AcceptStorageDestroy(r.Context(), headers, r.PathValue("id"), req)
		})
	})
	mux.HandleFunc("POST /api/fabric/storage-attachments/{id}/detach", func(w http.ResponseWriter, r *http.Request) {
		handleConfirmedMutation(w, r, func(headers service.MutationHeaders, req service.ConfirmRequest) (service.OperationReceipt, error) {
			return svc.AcceptAttachmentDetach(r.Context(), headers, r.PathValue("id"), req)
		})
	})
	mux.HandleFunc("GET /api/fabric/operations/{id}", func(w http.ResponseWriter, r *http.Request) {
		receipt, err := svc.Operation(r.Context(), r.PathValue("id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, receipt)
	})

	return requireOperatorToken(cfg.OperatorToken, mux)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func handleMutation[T any](w http.ResponseWriter, r *http.Request, accept func(context.Context, service.MutationHeaders, T) (service.OperationReceipt, error)) {
	headers, ok := mutationHeaders(w, r)
	if !ok {
		return
	}
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStatus(w, http.StatusBadRequest, "invalid_json")
		return
	}
	receipt, err := accept(r.Context(), headers, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSONStatus(w, http.StatusAccepted, receipt)
}

func handleConfirmedMutation(w http.ResponseWriter, r *http.Request, accept func(service.MutationHeaders, service.ConfirmRequest) (service.OperationReceipt, error)) {
	headers, ok := mutationHeaders(w, r)
	if !ok {
		return
	}
	var req service.ConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorStatus(w, http.StatusBadRequest, "invalid_json")
		return
	}
	receipt, err := accept(headers, req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSONStatus(w, http.StatusAccepted, receipt)
}

func mutationHeaders(w http.ResponseWriter, r *http.Request) (service.MutationHeaders, bool) {
	headers := service.MutationHeaders{
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		CorrelationID:  strings.TrimSpace(r.Header.Get("X-Correlation-Id")),
	}
	if headers.IdempotencyKey == "" || headers.CorrelationID == "" {
		writeErrorStatus(w, http.StatusBadRequest, "operation_headers_required")
		return service.MutationHeaders{}, false
	}
	return headers, true
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrStoreRequired):
		writeErrorStatus(w, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, service.ErrRequestedByMissing), errors.Is(err, service.ErrAccountIDMissing), errors.Is(err, service.ErrConfirmationNeeded):
		writeErrorStatus(w, http.StatusBadRequest, err.Error())
	default:
		writeErrorStatus(w, http.StatusInternalServerError, "fabric_operation_failed")
	}
}

func writeErrorStatus(w http.ResponseWriter, status int, code string) {
	writeJSONStatus(w, status, map[string]string{"error": code})
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
