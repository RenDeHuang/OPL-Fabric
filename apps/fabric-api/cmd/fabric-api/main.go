package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/config"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/fabricruntime"
	httpapi "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/http"
	fabrick8s "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/k8s"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/kubeconfig"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/orchestrator"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/tencentcloud"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/worker"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	store, err := openMigratedStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	if store != nil {
		defer store.Close()
	}
	cat := catalog.DefaultCatalog(catalog.Config{
		WorkspaceImage:  cfg.WorkspaceImage,
		WorkspaceDomain: cfg.WorkspaceDomain,
		StorageClass:    cfg.StorageClass,
	})
	svc := service.New(service.Config{
		Catalog:             cat,
		DatabaseURL:         cfg.DatabaseURL,
		OperatorToken:       cfg.OperatorToken,
		KubernetesNamespace: cfg.KubernetesNamespace,
		IngressClass:        cfg.IngressClass,
		ImagePullSecretName: cfg.ImagePullSecretName,
		WorkspaceImage:      cfg.WorkspaceImage,
		WorkspaceDomain:     cfg.WorkspaceDomain,
		StorageClass:        cfg.StorageClass,
		TencentTKERegion:    cfg.TencentTKERegion,
		TencentClusterID:    cfg.TencentDeployClusterID,
		TencentSecretID:     cfg.TencentMutationSecretID,
		TencentSecretKey:    cfg.TencentMutationSecretKey,
		TencentTCRRegistry:  cfg.TencentTCRRegistry,
		TencentTCRNamespace: cfg.TencentTCRNamespace,
		TencentTCRRegion:    cfg.TencentTCRRegion,
		Store:               store,
	})
	handler := httpapi.NewServer(svc, httpapi.Config{OperatorToken: cfg.OperatorToken})
	if cfg.WorkerEnabled == "true" {
		if err := startWorker(ctx, cfg, store); err != nil {
			log.Fatal(err)
		}
	}

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

func openMigratedStore(ctx context.Context, databaseURL string) (*postgres.Store, error) {
	if databaseURL == "" {
		return nil, nil
	}
	store, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		store.Close()
		return nil, err
	}
	return store, nil
}

func startWorker(ctx context.Context, cfg config.Config, store *postgres.Store) error {
	if store == nil {
		log.Printf("fabric worker requested but DATABASE_URL is not configured; worker disabled")
		return nil
	}
	restConfig, err := kubernetesRESTConfig(cfg)
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	runtime := fabricruntime.KubernetesRuntime{
		Provider: fabrick8s.Provider{
			Client:               client,
			Namespace:            cfg.KubernetesNamespace,
			WorkspaceImage:       cfg.WorkspaceImage,
			StorageClassName:     cfg.StorageClass,
			WorkspaceDomain:      cfg.WorkspaceDomain,
			IngressClassName:     cfg.IngressClass,
			WorkspaceWebUIPort:   parseInt32(cfg.WorkspaceWebUIPort, 3000),
			WorkspaceDataDir:     cfg.WorkspaceDataDir,
			WorkspaceProjectsDir: cfg.WorkspaceProjectsDir,
			CodexHome:            cfg.CodexHome,
			CodexModel:           cfg.CodexModel,
			CodexReasoningEffort: cfg.CodexReasoningEffort,
			CodexBaseURL:         cfg.CodexBaseURL,
			CodexAPIKey:          cfg.CodexAPIKey,
			CodexModelProvider:   cfg.CodexModelProvider,
			CodexProviderName:    cfg.CodexProviderName,
		},
		Capacity: capacityAdapter{provider: tencentcloud.NodePoolProvider{Config: tencentcloud.NodePoolResolverConfig{
			ClusterID:          cfg.TencentDeployClusterID,
			Region:             cfg.TencentTKERegion,
			SecretID:           cfg.TencentMutationSecretID,
			SecretKey:          cfg.TencentMutationSecretKey,
			LaunchConfigJSON:   cfg.TKENodePoolLaunchJSON,
			AutoscalingJSON:    cfg.TKENodePoolAutoscalingJSON,
			InstanceChargeType: cfg.TKEInstanceChargeType,
			DesiredPodNumber:   cfg.TKENodePoolDesiredPodNumber,
			MutationAllowed:    parseBool(cfg.TKEAllowNodePoolMutation),
		}}},
	}
	orch := orchestrator.Orchestrator{Store: store, Runtime: runtime}
	w := worker.Worker{
		Store:        store,
		Orchestrator: orch,
		Owner:        cfg.WorkerOwner,
		Interval:     parseDuration(cfg.WorkerInterval, 5*time.Second),
		LeaseTTL:     parseDuration(cfg.WorkerLeaseTTL, time.Minute),
		BatchSize:    parseInt(cfg.WorkerBatchSize, 10),
	}
	go func() {
		if err := w.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("fabric worker stopped: %v", err)
		}
	}()
	log.Printf("fabric worker enabled owner=%s interval=%s batch=%d", w.Owner, w.Interval, w.BatchSize)
	return nil
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseInt32(value string, fallback int32) int32 {
	parsed := parseInt(value, int(fallback))
	return int32(parsed)
}

func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func kubernetesRESTConfig(cfg config.Config) (*rest.Config, error) {
	return kubeconfig.LoadRESTConfig(cfg.TencentDeployKubeconfigRef)
}

type capacityAdapter struct {
	provider tencentcloud.NodePoolProvider
}

func (a capacityAdapter) EnsureNodePool(ctx context.Context, req fabricruntime.CapacityNodePoolRequest) (fabricruntime.CapacityNodePoolResult, error) {
	result, err := a.provider.EnsureNodePool(ctx, tencentcloud.NodePoolRequest{
		ComputeID:                 req.ComputeID,
		WorkspaceID:               req.WorkspaceID,
		RequestedComputeShapeJSON: req.RequestedComputeShapeJSON,
	})
	if err != nil {
		return fabricruntime.CapacityNodePoolResult{}, err
	}
	return fabricruntime.CapacityNodePoolResult{NodePoolID: result.NodePoolID}, nil
}

func (a capacityAdapter) VerifyNodePool(ctx context.Context, nodePoolID string) (bool, error) {
	return a.provider.VerifyNodePool(ctx, nodePoolID)
}

func (a capacityAdapter) DeleteNodePool(ctx context.Context, nodePoolID string) error {
	return a.provider.DeleteNodePool(ctx, nodePoolID)
}
