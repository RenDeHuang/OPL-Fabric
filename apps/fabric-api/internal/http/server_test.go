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

func TestCreateWorkspaceEndpointReturnsAcceptedOperation(t *testing.T) {
	store := &recordingStore{}
	cfg := testServiceConfig()
	cfg.Store = store
	svc := service.New(cfg)
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	body := `{"accountId":"acct-1","requestedBy":"user-1","workspaceName":"Lab","productPresetId":"basic","computeShape":{"cpu":2,"memoryGb":4},"storage":{"sizeGb":20},"isolationMode":"workspace_exclusive_cvm"}`
	req := httptest.NewRequest(http.MethodPost, "/api/fabric/workspaces", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "idem-workspace-delivery-1")
	req.Header.Set("X-Correlation-Id", "corr-workspace-delivery-1")
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
	if receipt.OperationID == "" || receipt.State != "accepted" || receipt.ResourceKind != "workspace" || receipt.ResourceID == "" {
		t.Fatalf("receipt mismatch: %+v", receipt)
	}
	if len(store.workspaces) != 1 || len(store.storageVolumes) != 1 || len(store.computeAllocations) != 1 || len(store.storageAttachments) != 1 || len(store.workspaceEntries) != 1 {
		t.Fatalf("aggregate rows missing: workspaces=%d storage=%d compute=%d attachments=%d entries=%d", len(store.workspaces), len(store.storageVolumes), len(store.computeAllocations), len(store.storageAttachments), len(store.workspaceEntries))
	}
	if store.workspaceReservations != 1 {
		t.Fatalf("workspace reservations = %d, want 1", store.workspaceReservations)
	}
}

func TestCreateWorkspaceEndpointDoesNotLeaveRowsWhenReservationFails(t *testing.T) {
	store := &recordingStore{reservationErr: postgres.ErrStoreNotOpen}
	cfg := testServiceConfig()
	cfg.Store = store
	svc := service.New(cfg)
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	body := `{"accountId":"acct-1","requestedBy":"user-1","workspaceName":"Lab"}`
	req := httptest.NewRequest(http.MethodPost, "/api/fabric/workspaces", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "idem-workspace-fail-1")
	req.Header.Set("X-Correlation-Id", "corr-workspace-fail-1")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	if len(store.workspaces) != 0 || len(store.storageVolumes) != 0 || len(store.computeAllocations) != 0 || len(store.storageAttachments) != 0 || len(store.workspaceEntries) != 0 || len(store.operations) != 0 {
		t.Fatalf("partial rows left behind: workspaces=%d storage=%d compute=%d attachments=%d entries=%d operations=%d", len(store.workspaces), len(store.storageVolumes), len(store.computeAllocations), len(store.storageAttachments), len(store.workspaceEntries), len(store.operations))
	}
}

func TestWorkspaceEndpointReturnsAggregateStatus(t *testing.T) {
	store := &recordingStore{}
	cfg := testServiceConfig()
	cfg.Store = store
	svc := service.New(cfg)
	server := NewServer(svc, Config{OperatorToken: "test-token"})

	body := `{"accountId":"acct-1","requestedBy":"user-1","workspaceName":"Lab","productPresetId":"basic","storage":{"sizeGb":20}}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/fabric/workspaces", strings.NewReader(body))
	createReq.Header.Set("Authorization", "Bearer test-token")
	createReq.Header.Set("Idempotency-Key", "idem-workspace-status-1")
	createReq.Header.Set("X-Correlation-Id", "corr-workspace-status-1")
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body=%s", createRec.Code, createRec.Body.String())
	}
	var receipt struct {
		ResourceID string `json:"resourceId"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/fabric/workspaces/"+receipt.ResourceID, nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getRec := httptest.NewRecorder()
	server.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d, body=%s", getRec.Code, http.StatusOK, getRec.Body.String())
	}
	var workspace struct {
		WorkspaceID string `json:"workspaceId"`
		State       string `json:"state"`
		Storage     struct {
			ID     string `json:"id"`
			State  string `json:"state"`
			SizeGB int    `json:"sizeGb"`
		} `json:"storage"`
		Entry struct {
			URL string `json:"url"`
		} `json:"entry"`
		OperationID string `json:"operationId"`
	}
	if err := json.NewDecoder(getRec.Body).Decode(&workspace); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	if workspace.WorkspaceID != receipt.ResourceID || workspace.State != "provisioning" || workspace.Storage.SizeGB != 20 || workspace.Entry.URL == "" || workspace.OperationID == "" {
		t.Fatalf("workspace mismatch: %+v", workspace)
	}
}

func TestResourceStatusEndpointsReturnDurableRows(t *testing.T) {
	store := &recordingStore{
		storageVolumes: map[string]postgres.StorageVolumeRow{
			"storage-1": {ID: "storage-1", State: "available", SizeGB: 20, ProviderRef: "pvc/storage-1", Retained: true},
		},
		computeAllocations: map[string]postgres.ComputeAllocationRow{
			"compute-1": {ID: "compute-1", State: "running", ProviderRef: "deployment/compute-1", RuntimeRef: "service/compute-1"},
		},
		storageAttachments: map[string]postgres.StorageAttachmentRow{
			"attach-1": {ID: "attach-1", ComputeAllocationID: "compute-1", StorageID: "storage-1", State: "attached", MountPath: "/data", ProviderRef: "deployment/compute-1:pvc/storage-1"},
		},
		workspaceEntries: map[string]postgres.WorkspaceEntryRow{
			"entry-1": {ID: "entry-1", WorkspaceID: "ws-1", AttachmentID: "attach-1", State: "ready", Host: "workspace.medopl.cn", Path: "/w/ws-1/", ServiceRef: "service/compute-1"},
		},
	}
	cfg := testServiceConfig()
	cfg.Store = store
	server := NewServer(service.New(cfg), Config{OperatorToken: "test-token"})
	cases := []struct {
		path string
		want string
	}{
		{"/api/fabric/storage-volumes/storage-1", `"sizeGb":20`},
		{"/api/fabric/compute-allocations/compute-1", `"runtimeRef":"service/compute-1"`},
		{"/api/fabric/storage-attachments/attach-1", `"mountPath":"/data"`},
		{"/api/fabric/workspace-entries/entry-1", `"serviceRef":"service/compute-1"`},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()

		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body=%s", tc.path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Fatalf("%s body = %s, want fragment %s", tc.path, rec.Body.String(), tc.want)
		}
	}
}

func TestResourceStatusEndpointsRequireAuth(t *testing.T) {
	cfg := testServiceConfig()
	cfg.Store = &recordingStore{}
	server := NewServer(service.New(cfg), Config{OperatorToken: "test-token"})
	req := httptest.NewRequest(http.MethodGet, "/api/fabric/storage-volumes/storage-1", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
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
		KubernetesNamespace: "oplfabric",
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
		Catalog:                    testCatalog(),
		DatabaseURL:                "postgres://user:pass@db:5432/opl_fabric",
		OperatorToken:              "test-token",
		KubernetesNamespace:        "oplfabric",
		IngressClass:               "qcloud",
		ImagePullSecretName:        "tcr-pull-secret",
		WorkspaceImage:             "ghcr.io/gaofeng21cn/one-person-lab-app:latest",
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
		Store:                      &recordingStore{},
	}
}

type recordingStore struct {
	operations            map[string]postgres.OperationRow
	storageVolumes        map[string]postgres.StorageVolumeRow
	computeAllocations    map[string]postgres.ComputeAllocationRow
	storageAttachments    map[string]postgres.StorageAttachmentRow
	workspaceEntries      map[string]postgres.WorkspaceEntryRow
	workspaces            map[string]postgres.WorkspaceRow
	workspaceReservations int
	reservationErr        error
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

func (s *recordingStore) CreateStorageVolume(_ context.Context, row postgres.StorageVolumeRow) error {
	if s.storageVolumes == nil {
		s.storageVolumes = map[string]postgres.StorageVolumeRow{}
	}
	s.storageVolumes[row.ID] = row
	return nil
}

func (s *recordingStore) CreateComputeAllocation(_ context.Context, row postgres.ComputeAllocationRow) error {
	if s.computeAllocations == nil {
		s.computeAllocations = map[string]postgres.ComputeAllocationRow{}
	}
	s.computeAllocations[row.ID] = row
	return nil
}

func (s *recordingStore) CreateStorageAttachment(_ context.Context, row postgres.StorageAttachmentRow) error {
	if s.storageAttachments == nil {
		s.storageAttachments = map[string]postgres.StorageAttachmentRow{}
	}
	s.storageAttachments[row.ID] = row
	return nil
}

func (s *recordingStore) CreateWorkspaceEntry(_ context.Context, row postgres.WorkspaceEntryRow) error {
	if s.workspaceEntries == nil {
		s.workspaceEntries = map[string]postgres.WorkspaceEntryRow{}
	}
	s.workspaceEntries[row.ID] = row
	return nil
}

func (s *recordingStore) CreateWorkspace(_ context.Context, row postgres.WorkspaceRow) error {
	if s.workspaces == nil {
		s.workspaces = map[string]postgres.WorkspaceRow{}
	}
	s.workspaces[row.ID] = row
	return nil
}

func (s *recordingStore) CreateWorkspaceReservation(ctx context.Context, reservation postgres.WorkspaceReservation) error {
	s.workspaceReservations++
	if s.reservationErr != nil {
		return s.reservationErr
	}
	if err := s.CreateOperation(ctx, reservation.Operation); err != nil {
		return err
	}
	if err := s.CreateStorageVolume(ctx, reservation.Storage); err != nil {
		return err
	}
	if err := s.CreateComputeAllocation(ctx, reservation.Compute); err != nil {
		return err
	}
	if err := s.CreateStorageAttachment(ctx, reservation.Attachment); err != nil {
		return err
	}
	if err := s.CreateWorkspaceEntry(ctx, reservation.Entry); err != nil {
		return err
	}
	return s.CreateWorkspace(ctx, reservation.Workspace)
}

func (s *recordingStore) GetWorkspace(_ context.Context, id string) (postgres.WorkspaceRow, error) {
	if row, ok := s.workspaces[id]; ok {
		return row, nil
	}
	return postgres.WorkspaceRow{}, postgres.ErrStoreNotOpen
}

func (s *recordingStore) GetStorageVolume(_ context.Context, id string) (postgres.StorageVolumeRow, error) {
	if row, ok := s.storageVolumes[id]; ok {
		return row, nil
	}
	return postgres.StorageVolumeRow{}, postgres.ErrStoreNotOpen
}

func (s *recordingStore) GetComputeAllocation(_ context.Context, id string) (postgres.ComputeAllocationRow, error) {
	if row, ok := s.computeAllocations[id]; ok {
		return row, nil
	}
	return postgres.ComputeAllocationRow{}, postgres.ErrStoreNotOpen
}

func (s *recordingStore) GetStorageAttachment(_ context.Context, id string) (postgres.StorageAttachmentRow, error) {
	if row, ok := s.storageAttachments[id]; ok {
		return row, nil
	}
	return postgres.StorageAttachmentRow{}, postgres.ErrStoreNotOpen
}

func (s *recordingStore) GetWorkspaceEntry(_ context.Context, id string) (postgres.WorkspaceEntryRow, error) {
	if row, ok := s.workspaceEntries[id]; ok {
		return row, nil
	}
	return postgres.WorkspaceEntryRow{}, postgres.ErrStoreNotOpen
}
