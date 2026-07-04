package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
)

func TestReadinessEndpoint(t *testing.T) {
	svc := service.New(testServiceConfig())
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	req := httptest.NewRequest(http.MethodGet, "/api/fabric/readiness", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var readiness struct {
		Ready           bool            `json:"ready"`
		Provider        string          `json:"provider"`
		MissingEnv      []string        `json:"missingEnv"`
		ResourceCatalog catalog.Catalog `json:"resourceCatalog"`
		Blockers        []string        `json:"blockers"`
		RepairHints     []string        `json:"repairHints"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&readiness); err != nil {
		t.Fatalf("decode readiness: %v", err)
	}

	if !readiness.Ready {
		t.Fatal("ready = false, want true")
	}
	if readiness.Provider != "tencent-tke" {
		t.Fatalf("provider = %q, want tencent-tke", readiness.Provider)
	}
	if readiness.ResourceCatalog.SchemaVersion != 2 {
		t.Fatalf("resourceCatalog schemaVersion = %d, want 2", readiness.ResourceCatalog.SchemaVersion)
	}
	if len(readiness.ResourceCatalog.ProductPresets) != 3 {
		t.Fatalf("resourceCatalog product preset count = %d, want 3", len(readiness.ResourceCatalog.ProductPresets))
	}
	if readiness.ResourceCatalog.ProductPresets[0].ID != "basic" {
		t.Fatalf("resourceCatalog first product preset ID = %q, want basic", readiness.ResourceCatalog.ProductPresets[0].ID)
	}
	if len(readiness.MissingEnv) != 0 {
		t.Fatalf("missingEnv = %v, want empty", readiness.MissingEnv)
	}
	if len(readiness.Blockers) != 0 {
		t.Fatalf("blockers = %v, want empty", readiness.Blockers)
	}
	if len(readiness.RepairHints) != 0 {
		t.Fatalf("repairHints = %v, want empty", readiness.RepairHints)
	}
}

func TestCreateStorageVolumeEndpointReturnsAcceptedOperation(t *testing.T) {
	svc := service.New(testServiceConfig())
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	body := `{"accountId":"acct-1","requestedBy":"user-1","productPresetId":"basic","sizeGb":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/fabric/storage-volumes", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "idem-storage-1")
	req.Header.Set("X-Correlation-Id", "corr-storage-1")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var receipt struct {
		OperationID  string `json:"operationId"`
		State        string `json:"state"`
		ResourceKind string `json:"resourceKind"`
		ResourceID   string `json:"resourceId"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt.OperationID == "" || receipt.State != "accepted" || receipt.ResourceKind != "storage_volume" || receipt.ResourceID == "" {
		t.Fatalf("receipt mismatch: %+v", receipt)
	}
}

func TestMutatingEndpointRequiresOperationHeaders(t *testing.T) {
	svc := service.New(testServiceConfig())
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	req := httptest.NewRequest(http.MethodPost, "/api/fabric/storage-volumes", strings.NewReader(`{"accountId":"acct-1","requestedBy":"user-1","sizeGb":10}`))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateWorkspaceEntryPersistsGatewayPath(t *testing.T) {
	store := &recordingStore{}
	cfg := testServiceConfig()
	cfg.Store = store
	svc := service.New(cfg)
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	body := `{"accountId":"acct-1","requestedBy":"user-1","workspaceName":"Lab","attachmentId":"attach-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/fabric/workspace-entries", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "idem-workspace-1")
	req.Header.Set("X-Correlation-Id", "corr-workspace-1")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if len(store.workspaceEntries) != 1 {
		t.Fatalf("workspace entries = %d, want 1", len(store.workspaceEntries))
	}
	for _, entry := range store.workspaceEntries {
		if !strings.HasPrefix(entry.Path, "/w/") || !strings.HasSuffix(entry.Path, "/") {
			t.Fatalf("workspace path = %q, want /w/<workspaceId>/", entry.Path)
		}
	}
}

func TestReadinessReportsMissingRuntimeConfig(t *testing.T) {
	svc := service.New(service.Config{Catalog: testCatalog()})
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	req := httptest.NewRequest(http.MethodGet, "/api/fabric/readiness", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var readiness struct {
		Ready       bool     `json:"ready"`
		MissingEnv  []string `json:"missingEnv"`
		Blockers    []string `json:"blockers"`
		RepairHints []string `json:"repairHints"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&readiness); err != nil {
		t.Fatalf("decode readiness: %v", err)
	}

	if readiness.Ready {
		t.Fatal("ready = true, want false")
	}
	if len(readiness.MissingEnv) == 0 {
		t.Fatal("missingEnv should report missing runtime config")
	}
	if len(readiness.Blockers) == 0 {
		t.Fatal("blockers should report missing runtime config")
	}
	if len(readiness.RepairHints) == 0 {
		t.Fatal("repairHints should explain missing runtime config")
	}
}

func TestCatalogEndpoint(t *testing.T) {
	cat := testCatalog()
	svc := service.New(service.Config{
		Catalog:             cat,
		DatabaseURL:         "postgres://user:pass@db:5432/opl_fabric",
		OperatorToken:       "test-token",
		KubernetesNamespace: "opl-fabric",
		IngressClass:        "qcloud",
		ImagePullSecretName: "tcr-pull-secret",
	})
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	req := httptest.NewRequest(http.MethodGet, "/api/fabric/catalog", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var got catalog.Catalog
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}

	if got.SchemaVersion != cat.SchemaVersion {
		t.Fatalf("schemaVersion = %d, want %d", got.SchemaVersion, cat.SchemaVersion)
	}
	if got.Owner != cat.Owner {
		t.Fatalf("owner = %q, want %q", got.Owner, cat.Owner)
	}
	if len(got.ProductPresets) != len(cat.ProductPresets) {
		t.Fatalf("product preset count = %d, want %d", len(got.ProductPresets), len(cat.ProductPresets))
	}
	if got.ProductPresets[0].ID != "basic" {
		t.Fatalf("first product preset ID = %q, want basic", got.ProductPresets[0].ID)
	}
	if got.StorageClasses[0].StorageClassName != "cbs" {
		t.Fatalf("storage class = %q, want cbs", got.StorageClasses[0].StorageClassName)
	}
}

func TestServerRequiresOperatorToken(t *testing.T) {
	svc := service.New(testServiceConfig())
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	for _, tc := range []struct {
		name          string
		authorization string
	}{
		{name: "missing"},
		{name: "wrong", authorization: "Bearer wrong-token"},
		{name: "wrong scheme", authorization: "Basic test-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/fabric/readiness", nil)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestServerRejectsRequestsWhenOperatorTokenIsNotConfigured(t *testing.T) {
	svc := service.New(testServiceConfig())
	server := NewServer(svc, Config{})

	req := httptest.NewRequest(http.MethodGet, "/api/fabric/readiness", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestServerMethodAndPathHandling(t *testing.T) {
	svc := service.New(testServiceConfig())
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	methodReq := httptest.NewRequest(http.MethodPost, "/api/fabric/readiness", nil)
	methodReq.Header.Set("Authorization", "Bearer test-token")
	methodRec := httptest.NewRecorder()
	server.ServeHTTP(methodRec, methodReq)
	if methodRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d, want %d", methodRec.Code, http.StatusMethodNotAllowed)
	}

	pathReq := httptest.NewRequest(http.MethodGet, "/api/fabric/missing", nil)
	pathReq.Header.Set("Authorization", "Bearer test-token")
	pathRec := httptest.NewRecorder()
	server.ServeHTTP(pathRec, pathReq)
	if pathRec.Code != http.StatusNotFound {
		t.Fatalf("path status = %d, want %d", pathRec.Code, http.StatusNotFound)
	}
}

func testCatalog() catalog.Catalog {
	return catalog.DefaultCatalog(catalog.Config{
		WorkspaceImage:  "ghcr.io/gaofeng21cn/one-person-lab-app:latest",
		WorkspaceDomain: "workspace.medopl.cn",
		StorageClass:    "cbs",
	})
}

func testServiceConfig() service.Config {
	return service.Config{
		Catalog:             testCatalog(),
		DatabaseURL:         "postgres://user:pass@db:5432/opl_fabric",
		OperatorToken:       "test-token",
		KubernetesNamespace: "opl-fabric",
		IngressClass:        "qcloud",
		ImagePullSecretName: "tcr-pull-secret",
		WorkspaceImage:      "ghcr.io/gaofeng21cn/one-person-lab-app:latest",
		WorkspaceDomain:     "workspace.medopl.cn",
		StorageClass:        "cbs",
		TencentTKERegion:    "ap-guangzhou",
		TencentClusterID:    "cls-example",
		TencentSecretID:     "secret-id",
		TencentSecretKey:    "secret-key",
		TencentTCRRegistry:  "registry.example.com",
		TencentTCRNamespace: "opl",
		TencentTCRRegion:    "ap-guangzhou",
		Store:               &recordingStore{},
	}
}

type recordingStore struct {
	operations       map[string]postgres.OperationRow
	workspaceEntries map[string]postgres.WorkspaceEntryRow
}

func (s *recordingStore) CreateOperation(_ context.Context, row postgres.OperationRow) error {
	if s.operations == nil {
		s.operations = map[string]postgres.OperationRow{}
	}
	s.operations[row.ID] = row
	return nil
}

func (s *recordingStore) GetOperation(_ context.Context, id string) (postgres.OperationRow, error) {
	if row, ok := s.operations[id]; ok {
		return row, nil
	}
	return postgres.OperationRow{}, postgres.ErrStoreNotOpen
}

func (s *recordingStore) CreateStorageVolume(context.Context, postgres.StorageVolumeRow) error {
	return nil
}

func (s *recordingStore) CreateComputeResource(context.Context, postgres.ComputeResourceRow) error {
	return nil
}

func (s *recordingStore) CreateStorageAttachment(context.Context, postgres.StorageAttachmentRow) error {
	return nil
}

func (s *recordingStore) CreateWorkspaceEntry(_ context.Context, row postgres.WorkspaceEntryRow) error {
	if s.workspaceEntries == nil {
		s.workspaceEntries = map[string]postgres.WorkspaceEntryRow{}
	}
	s.workspaceEntries[row.ID] = row
	return nil
}
