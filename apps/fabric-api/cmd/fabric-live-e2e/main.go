package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/catalog"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/config"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/fabricruntime"
	fabrick8s "github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/k8s"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/kubeconfig"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/orchestrator"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/service"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/staging"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/tencentcloud"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/worker"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), durationEnv("OPL_LIVE_E2E_TIMEOUT", 30*time.Minute))
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg := config.Load()
	gate := staging.EvaluateGate(staging.Config{
		DatabaseURL:                cfg.DatabaseURL,
		OperatorToken:              cfg.OperatorToken,
		KubeconfigRef:              cfg.TencentDeployKubeconfigRef,
		Namespace:                  cfg.KubernetesNamespace,
		StorageClass:               cfg.StorageClass,
		IngressClass:               cfg.IngressClass,
		WorkspaceDomain:            cfg.WorkspaceDomain,
		WorkspaceImage:             cfg.WorkspaceImage,
		ImagePullSecretName:        cfg.ImagePullSecretName,
		TencentClusterID:           cfg.TencentDeployClusterID,
		TencentRegion:              cfg.TencentTKERegion,
		TencentSecretID:            cfg.TencentMutationSecretID,
		TencentSecretKey:           cfg.TencentMutationSecretKey,
		TencentTCRRegistry:         cfg.TencentTCRRegistry,
		TencentTCRNamespace:        cfg.TencentTCRNamespace,
		TencentTCRRegion:           cfg.TencentTCRRegion,
		TencentCVMSubnetIDs:        cfg.TencentCVMSubnetIDs,
		TencentCVMSecurityGroupIDs: cfg.TencentCVMSecurityGroupIDs,
		TencentCVMSystemDiskType:   cfg.TencentCVMSystemDiskType,
		TencentCVMSystemDiskSizeGB: cfg.TencentCVMSystemDiskSizeGB,
		AllowNodePoolMutation:      boolEnv("OPL_TKE_ALLOW_NODEPOOL_MUTATION"),
		AllowStagingE2E:            boolEnv("OPL_STAGING_E2E_ALLOW_LIVE"),
		WorkerEnabled:              boolEnv("OPL_FABRIC_WORKER_ENABLED"),
	})
	if !gate.Ready || gate.Mode != "ready_for_live" {
		return fmt.Errorf("live gate blocked mode=%s missing=%v blockers=%v", gate.Mode, gate.Missing, gate.Blockers)
	}
	log.Printf("live gate ready mode=%s", gate.Mode)

	var k8sClient *kubernetes.Clientset
	if boolEnv("OPL_LIVE_E2E_SKIP_K8S_CHECK") {
		log.Printf("kubernetes check skipped by OPL_LIVE_E2E_SKIP_K8S_CHECK=true")
	} else {
		log.Printf("loading kubeconfig")
		restConfig, err := kubeconfig.LoadRESTConfig(cfg.TencentDeployKubeconfigRef)
		if err != nil {
			return err
		}
		k8sClient, err = kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		log.Printf("verifying kubernetes inputs")
		if err := verifyKubernetesInputs(ctx, k8sClient, cfg); err != nil {
			return err
		}
	}
	log.Printf("verifying tencent inputs")
	if err := verifyTencentInputs(ctx, cfg); err != nil {
		return err
	}
	if boolEnv("OPL_LIVE_E2E_CHECK_ONLY") {
		if boolEnv("OPL_LIVE_E2E_SKIP_POSTGRES_CHECK") {
			log.Printf("postgres check skipped by OPL_LIVE_E2E_SKIP_POSTGRES_CHECK=true")
		} else {
			if err := verifyPostgres(ctx, cfg); err != nil {
				return err
			}
		}
		log.Printf("live e2e check-only passed cluster=%s region=%s namespace=%s image=%s storageClass=%s ingressClass=%s", cfg.TencentDeployClusterID, cfg.TencentTKERegion, cfg.KubernetesNamespace, cfg.WorkspaceImage, cfg.StorageClass, cfg.IngressClass)
		return nil
	}
	if k8sClient == nil {
		return errors.New("kubernetes client is required for live e2e execution")
	}

	log.Printf("opening postgres store")
	store, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()
	log.Printf("running postgres migration")
	if err := store.Migrate(ctx); err != nil {
		return err
	}

	runtime := fabricruntime.KubernetesRuntime{
		Provider: fabrick8s.Provider{
			Client:               k8sClient,
			Namespace:            cfg.KubernetesNamespace,
			WorkspaceImage:       cfg.WorkspaceImage,
			ImagePullSecretName:  cfg.ImagePullSecretName,
			StorageClassName:     cfg.StorageClass,
			WorkspaceDomain:      cfg.WorkspaceDomain,
			IngressClassName:     cfg.IngressClass,
			WorkspaceWebUIPort:   int32Env("OPL_WORKSPACE_WEBUI_PORT", 3000),
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
			SubnetIDs:          cfg.TencentCVMSubnetIDs,
			SecurityGroupIDs:   cfg.TencentCVMSecurityGroupIDs,
			SystemDiskType:     cfg.TencentCVMSystemDiskType,
			SystemDiskSizeGB:   cfg.TencentCVMSystemDiskSizeGB,
			InstanceChargeType: cfg.TKEInstanceChargeType,
			DesiredPodNumber:   cfg.TKENodePoolDesiredPodNumber,
			MutationAllowed:    boolEnv("OPL_TKE_ALLOW_NODEPOOL_MUTATION"),
		}}},
	}
	orch := orchestrator.Orchestrator{Store: store, Runtime: runtime}
	w := worker.Worker{Store: store, Orchestrator: orch, Owner: "fabric-live-e2e", BatchSize: 10, LeaseTTL: time.Minute}
	svc := service.New(service.Config{
		Catalog:                    catalog.DefaultCatalog(catalog.Config{WorkspaceImage: cfg.WorkspaceImage, WorkspaceDomain: cfg.WorkspaceDomain, StorageClass: cfg.StorageClass}),
		DatabaseURL:                cfg.DatabaseURL,
		OperatorToken:              cfg.OperatorToken,
		KubernetesNamespace:        cfg.KubernetesNamespace,
		IngressClass:               cfg.IngressClass,
		ImagePullSecretName:        cfg.ImagePullSecretName,
		WorkspaceImage:             cfg.WorkspaceImage,
		WorkspaceDomain:            cfg.WorkspaceDomain,
		StorageClass:               cfg.StorageClass,
		TencentTKERegion:           cfg.TencentTKERegion,
		TencentClusterID:           cfg.TencentDeployClusterID,
		TencentSecretID:            cfg.TencentMutationSecretID,
		TencentSecretKey:           cfg.TencentMutationSecretKey,
		TencentTCRRegistry:         cfg.TencentTCRRegistry,
		TencentTCRNamespace:        cfg.TencentTCRNamespace,
		TencentTCRRegion:           cfg.TencentTCRRegion,
		TencentCVMSubnetIDs:        cfg.TencentCVMSubnetIDs,
		TencentCVMSecurityGroupIDs: cfg.TencentCVMSecurityGroupIDs,
		Store:                      store,
	})

	idem := stringEnv("OPL_LIVE_E2E_IDEMPOTENCY_KEY", "fabric-live-"+time.Now().UTC().Format("20060102T150405"))
	log.Printf("live e2e target cluster=%s region=%s namespace=%s image=%s storageClass=%s ingressClass=%s idempotency=%s", cfg.TencentDeployClusterID, cfg.TencentTKERegion, cfg.KubernetesNamespace, cfg.WorkspaceImage, cfg.StorageClass, cfg.IngressClass, idem)
	receipt, err := svc.AcceptWorkspace(ctx, service.MutationHeaders{IdempotencyKey: idem, CorrelationID: "corr-" + idem}, service.CreateWorkspaceRequest{
		AccountID:            stringEnv("OPL_LIVE_E2E_ACCOUNT_ID", "acct-live-e2e"),
		RequestedBy:          stringEnv("OPL_LIVE_E2E_REQUESTED_BY", "fabric-live-e2e"),
		WorkspaceName:        stringEnv("OPL_LIVE_E2E_WORKSPACE_NAME", "Fabric Live E2E"),
		ProductPresetID:      stringEnv("OPL_LIVE_E2E_PRODUCT_PRESET_ID", "basic"),
		ComputeShape:         map[string]any{"cpu": intEnv("OPL_LIVE_E2E_CPU", 2), "memoryGb": intEnv("OPL_LIVE_E2E_MEMORY_GB", 4)},
		ProviderInstanceType: os.Getenv("OPL_LIVE_E2E_PROVIDER_INSTANCE_TYPE"),
		CapacityPoolID:       stringEnv("OPL_LIVE_E2E_CAPACITY_POOL_ID", "dedicated-nodepool-template"),
		IsolationMode:        stringEnv("OPL_LIVE_E2E_ISOLATION_MODE", "dedicated_nodepool"),
		Storage: struct {
			SizeGB int `json:"sizeGb"`
		}{SizeGB: intEnv("OPL_LIVE_E2E_STORAGE_GB", 20)},
	})
	if err != nil {
		return err
	}
	if err := waitOperation(ctx, store, w, receipt.OperationID); err != nil {
		return err
	}
	workspace, err := svc.Workspace(ctx, receipt.ResourceID)
	if err != nil {
		return err
	}
	if workspace.State != "running" {
		return fmt.Errorf("workspace state=%s, want running", workspace.State)
	}
	if err := verifyWorkspaceObjects(ctx, k8sClient, cfg.KubernetesNamespace, store, workspace.WorkspaceID); err != nil {
		return err
	}
	log.Printf("workspace running id=%s url=%s operation=%s", workspace.WorkspaceID, workspace.Entry.URL, workspace.OperationID)

	destroyReceipt, err := svc.AcceptComputeDestroy(ctx, service.MutationHeaders{IdempotencyKey: idem + "-destroy-compute", CorrelationID: "corr-" + idem + "-destroy-compute"}, workspace.Compute.ID, service.ConfirmRequest{RequestedBy: "fabric-live-e2e", Confirm: true})
	if err != nil {
		return err
	}
	if err := waitOperation(ctx, store, w, destroyReceipt.OperationID); err != nil {
		return err
	}
	if err := verifyStorageRetained(ctx, k8sClient, cfg.KubernetesNamespace, store, workspace.WorkspaceID); err != nil {
		return err
	}
	log.Printf("compute destroyed and storage retained workspace=%s compute=%s", workspace.WorkspaceID, workspace.Compute.ID)

	rebuildOperationID := "op-" + idem + "-rebuild-compute"
	if err := store.CreateOperation(ctx, postgres.OperationRow{
		ID:             rebuildOperationID,
		CorrelationID:  "corr-" + idem + "-rebuild-compute",
		IdempotencyKey: idem + "-rebuild-compute",
		RequestedBy:    "fabric-live-e2e",
		ResourceID:     workspace.Compute.ID,
		ResourceKind:   "compute_resource",
		State:          "accepted",
	}); err != nil {
		return err
	}
	if err := waitOperation(ctx, store, w, rebuildOperationID); err != nil {
		return err
	}
	if err := verifyWorkspaceObjects(ctx, k8sClient, cfg.KubernetesNamespace, store, workspace.WorkspaceID); err != nil {
		return err
	}
	log.Printf("compute rebuilt and storage remounted workspace=%s compute=%s", workspace.WorkspaceID, workspace.Compute.ID)
	return nil
}

func verifyKubernetesInputs(ctx context.Context, client kubernetes.Interface, cfg config.Config) error {
	if boolEnv("OPL_LIVE_E2E_BOOTSTRAP_NAMESPACE") {
		if err := bootstrapKubernetesNamespace(ctx, client, cfg); err != nil {
			return err
		}
	}
	if _, err := client.CoreV1().Namespaces().Get(ctx, cfg.KubernetesNamespace, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("namespace %q: %w", cfg.KubernetesNamespace, err)
	}
	if _, err := client.StorageV1().StorageClasses().Get(ctx, cfg.StorageClass, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("storage class %q: %w", cfg.StorageClass, err)
	}
	if _, err := client.NetworkingV1().IngressClasses().Get(ctx, cfg.IngressClass, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("ingress class %q: %w", cfg.IngressClass, err)
	}
	if _, err := client.CoreV1().Secrets(cfg.KubernetesNamespace).Get(ctx, cfg.ImagePullSecretName, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("image pull secret %q: %w", cfg.ImagePullSecretName, err)
	}
	return nil
}

func bootstrapKubernetesNamespace(ctx context.Context, client kubernetes.Interface, cfg config.Config) error {
	if cfg.KubernetesNamespace == "" {
		return errors.New("kubernetes namespace is required")
	}
	if _, err := client.CoreV1().Namespaces().Get(ctx, cfg.KubernetesNamespace, metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("namespace %q: %w", cfg.KubernetesNamespace, err)
		}
		log.Printf("creating namespace %q", cfg.KubernetesNamespace)
		if _, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: cfg.KubernetesNamespace,
				Labels: map[string]string{
					"app.kubernetes.io/part-of": "opl-fabric",
				},
			},
		}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create namespace %q: %w", cfg.KubernetesNamespace, err)
		}
	}
	if err := ensureFabricServiceAccount(ctx, client, cfg.KubernetesNamespace); err != nil {
		return err
	}
	if err := ensureFabricRole(ctx, client, cfg.KubernetesNamespace); err != nil {
		return err
	}
	if err := ensureFabricRoleBinding(ctx, client, cfg.KubernetesNamespace); err != nil {
		return err
	}
	if boolEnv("OPL_LIVE_E2E_BOOTSTRAP_IMAGE_PULL_SECRET") {
		if err := ensureImagePullSecret(ctx, client, cfg.KubernetesNamespace, cfg.ImagePullSecretName); err != nil {
			return err
		}
	}
	return nil
}

func ensureFabricServiceAccount(ctx context.Context, client kubernetes.Interface, namespace string) error {
	const name = "opl-fabric-api"
	if _, err := client.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("service account %q: %w", name, err)
	}
	_, err := client.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: fabricControlPlaneLabels()},
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create service account %q: %w", name, err)
	}
	return nil
}

func ensureFabricRole(ctx context.Context, client kubernetes.Interface, namespace string) error {
	const name = "opl-fabric-api"
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: fabricControlPlaneLabels()},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{"apps"}, Resources: []string{"deployments"}, Verbs: []string{"create", "get", "update", "delete"}},
			{APIGroups: []string{""}, Resources: []string{"services"}, Verbs: []string{"create", "get", "delete"}},
			{APIGroups: []string{""}, Resources: []string{"persistentvolumeclaims", "secrets"}, Verbs: []string{"create", "get", "delete"}},
			{APIGroups: []string{"networking.k8s.io"}, Resources: []string{"ingresses"}, Verbs: []string{"create", "get", "update"}},
		},
	}
	existing, err := client.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		role.ResourceVersion = existing.ResourceVersion
		if _, err := client.RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update role %q: %w", name, err)
		}
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("role %q: %w", name, err)
	}
	if _, err := client.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create role %q: %w", name, err)
	}
	return nil
}

func ensureFabricRoleBinding(ctx context.Context, client kubernetes.Interface, namespace string) error {
	const name = "opl-fabric-api"
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: fabricControlPlaneLabels()},
		Subjects: []rbacv1.Subject{{
			Kind: "ServiceAccount",
			Name: name,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
	}
	existing, err := client.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		binding.ResourceVersion = existing.ResourceVersion
		if _, err := client.RbacV1().RoleBindings(namespace).Update(ctx, binding, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update role binding %q: %w", name, err)
		}
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("role binding %q: %w", name, err)
	}
	if _, err := client.RbacV1().RoleBindings(namespace).Create(ctx, binding, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create role binding %q: %w", name, err)
	}
	return nil
}

func ensureImagePullSecret(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	if namespace == "" || name == "" {
		return errors.New("image pull secret namespace and name are required")
	}
	if _, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("image pull secret %q: %w", name, err)
	}
	sourceNamespace := strings.TrimSpace(os.Getenv("OPL_LIVE_E2E_IMAGE_PULL_SECRET_SOURCE_NAMESPACE"))
	if sourceNamespace == "" || sourceNamespace == namespace {
		return nil
	}
	source, err := client.CoreV1().Secrets(sourceNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("source image pull secret %s/%s: %w", sourceNamespace, name, err)
	}
	copy := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: fabricControlPlaneLabels()},
		Type:       source.Type,
		Data:       cloneByteMap(source.Data),
	}
	if _, err := client.CoreV1().Secrets(namespace).Create(ctx, copy, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("copy image pull secret %s/%s to %s/%s: %w", sourceNamespace, name, namespace, name, err)
	}
	return nil
}

func fabricControlPlaneLabels() map[string]string {
	return map[string]string{"app.kubernetes.io/name": "opl-fabric-api"}
}

func cloneByteMap(values map[string][]byte) map[string][]byte {
	if values == nil {
		return nil
	}
	result := make(map[string][]byte, len(values))
	for key, value := range values {
		result[key] = append([]byte(nil), value...)
	}
	return result
}

func verifyPostgres(ctx context.Context, cfg config.Config) error {
	log.Printf("opening postgres store")
	store, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()
	log.Printf("running postgres migration")
	return store.Migrate(ctx)
}

func verifyTencentInputs(ctx context.Context, cfg config.Config) error {
	nodePoolID := os.Getenv("OPL_LIVE_E2E_VERIFY_NODEPOOL_ID")
	if nodePoolID == "" {
		return nil
	}
	provider := tencentcloud.NodePoolProvider{Config: tencentcloud.NodePoolResolverConfig{
		ClusterID:          cfg.TencentDeployClusterID,
		Region:             cfg.TencentTKERegion,
		SecretID:           cfg.TencentMutationSecretID,
		SecretKey:          cfg.TencentMutationSecretKey,
		SubnetIDs:          cfg.TencentCVMSubnetIDs,
		SecurityGroupIDs:   cfg.TencentCVMSecurityGroupIDs,
		SystemDiskType:     cfg.TencentCVMSystemDiskType,
		SystemDiskSizeGB:   cfg.TencentCVMSystemDiskSizeGB,
		InstanceChargeType: cfg.TKEInstanceChargeType,
		DesiredPodNumber:   cfg.TKENodePoolDesiredPodNumber,
		MutationAllowed:    boolEnv("OPL_TKE_ALLOW_NODEPOOL_MUTATION"),
	}}
	verified, err := provider.VerifyNodePool(ctx, nodePoolID)
	if err != nil {
		return fmt.Errorf("tencent verify nodepool %q: %w", nodePoolID, err)
	}
	if !verified {
		return fmt.Errorf("tencent verify nodepool %q: not found or not ready", nodePoolID)
	}
	return nil
}

func waitOperation(ctx context.Context, store *postgres.Store, w worker.Worker, operationID string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			return err
		}
		op, err := store.GetOperation(ctx, operationID)
		if err != nil {
			return err
		}
		switch op.State {
		case "succeeded":
			return nil
		case "failed":
			return fmt.Errorf("operation %s failed after %d attempts: %s", operationID, op.Attempts, op.LastError)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func verifyWorkspaceObjects(ctx context.Context, client kubernetes.Interface, namespace string, store *postgres.Store, workspaceID string) error {
	workspace, err := store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	storage, err := store.GetStorageVolume(ctx, workspace.StorageID)
	if err != nil {
		return err
	}
	compute, err := store.GetComputeResource(ctx, workspace.ComputeID)
	if err != nil {
		return err
	}
	attachment, err := store.GetStorageAttachment(ctx, workspace.AttachmentID)
	if err != nil {
		return err
	}
	entry, err := store.GetWorkspaceEntry(ctx, workspace.EntryID)
	if err != nil {
		return err
	}
	if storage.ProviderRef == "" || compute.ProviderRef == "" || compute.RuntimeRef == "" || attachment.ProviderRef == "" || entry.ServiceRef == "" {
		return fmt.Errorf("provider refs incomplete storage=%q compute=%q runtime=%q attachment=%q entryService=%q", storage.ProviderRef, compute.ProviderRef, compute.RuntimeRef, attachment.ProviderRef, entry.ServiceRef)
	}
	if _, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, resourceName(storage.ProviderRef), metav1.GetOptions{}); err != nil {
		return fmt.Errorf("pvc %q: %w", storage.ProviderRef, err)
	}
	if _, err := client.AppsV1().Deployments(namespace).Get(ctx, resourceName(compute.ProviderRef), metav1.GetOptions{}); err != nil {
		return fmt.Errorf("deployment %q: %w", compute.ProviderRef, err)
	}
	if _, err := client.CoreV1().Services(namespace).Get(ctx, resourceName(compute.RuntimeRef), metav1.GetOptions{}); err != nil {
		return fmt.Errorf("service %q: %w", compute.RuntimeRef, err)
	}
	if _, err := client.NetworkingV1().Ingresses(namespace).Get(ctx, "opl-fabric-workspace-gateway", metav1.GetOptions{}); err != nil {
		return fmt.Errorf("workspace ingress: %w", err)
	}
	if err := waitDeploymentAvailable(ctx, client, namespace, resourceName(compute.ProviderRef)); err != nil {
		return err
	}
	return nil
}

func verifyStorageRetained(ctx context.Context, client kubernetes.Interface, namespace string, store *postgres.Store, workspaceID string) error {
	workspace, err := store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	storage, err := store.GetStorageVolume(ctx, workspace.StorageID)
	if err != nil {
		return err
	}
	compute, err := store.GetComputeResource(ctx, workspace.ComputeID)
	if err != nil {
		return err
	}
	if compute.State != "destroyed" || compute.ProviderRef != "" || compute.RuntimeRef != "" || compute.NodePoolID != "" {
		return fmt.Errorf("compute not cleanly destroyed: %+v", compute)
	}
	if !storage.Retained || storage.ProviderRef == "" {
		return fmt.Errorf("storage not retained: %+v", storage)
	}
	if _, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, resourceName(storage.ProviderRef), metav1.GetOptions{}); err != nil {
		return fmt.Errorf("retained pvc %q: %w", storage.ProviderRef, err)
	}
	return nil
}

func waitDeploymentAvailable(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if deploymentAvailable(deploy) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("deployment %s did not become available: %w\n%s", name, ctx.Err(), deploymentDiagnostics(ctx, client, namespace, deploy))
		case <-ticker.C:
		}
	}
}

func deploymentDiagnostics(ctx context.Context, client kubernetes.Interface, namespace string, deploy *appsv1.Deployment) string {
	lines := []string{
		fmt.Sprintf("deployment_status name=%s namespace=%s replicas=%d updated=%d ready=%d available=%d observedGeneration=%d generation=%d selector=%s",
			deploy.Name,
			deploy.Namespace,
			deploy.Status.Replicas,
			deploy.Status.UpdatedReplicas,
			deploy.Status.ReadyReplicas,
			deploy.Status.AvailableReplicas,
			deploy.Status.ObservedGeneration,
			deploy.Generation,
			labels.Set(deploy.Spec.Selector.MatchLabels).String(),
		),
		fmt.Sprintf("pod_node_selector %s", labels.Set(deploy.Spec.Template.Spec.NodeSelector).String()),
	}
	for _, condition := range deploy.Status.Conditions {
		lines = append(lines, fmt.Sprintf("condition %s=%s reason=%s message=%s", condition.Type, condition.Status, condition.Reason, condition.Message))
	}
	pods, err := client.CoreV1().Pods(namespace).List(contextWithoutCancellation(ctx), metav1.ListOptions{LabelSelector: labels.Set(deploy.Spec.Selector.MatchLabels).String()})
	if err != nil {
		lines = append(lines, fmt.Sprintf("pod_list_error %v", err))
		return strings.Join(lines, "\n")
	}
	sort.Slice(pods.Items, func(i, j int) bool { return pods.Items[i].Name < pods.Items[j].Name })
	for _, pod := range pods.Items {
		lines = append(lines, fmt.Sprintf("pod name=%s phase=%s node=%s reason=%s message=%s", pod.Name, pod.Status.Phase, pod.Spec.NodeName, pod.Status.Reason, pod.Status.Message))
		for _, condition := range pod.Status.Conditions {
			if condition.Status != corev1.ConditionTrue {
				lines = append(lines, fmt.Sprintf("pod_condition pod=%s %s=%s reason=%s message=%s", pod.Name, condition.Type, condition.Status, condition.Reason, condition.Message))
			}
		}
		for _, status := range pod.Status.ContainerStatuses {
			switch {
			case status.State.Waiting != nil:
				lines = append(lines, fmt.Sprintf("container %s waiting=%s message=%s", status.Name, status.State.Waiting.Reason, status.State.Waiting.Message))
			case status.State.Terminated != nil:
				lines = append(lines, fmt.Sprintf("container %s terminated=%s exitCode=%d message=%s", status.Name, status.State.Terminated.Reason, status.State.Terminated.ExitCode, status.State.Terminated.Message))
			default:
				lines = append(lines, fmt.Sprintf("container %s ready=%t restarts=%d", status.Name, status.Ready, status.RestartCount))
			}
		}
		lines = append(lines, podEvents(ctx, client, namespace, pod.Name)...)
	}
	if len(pods.Items) == 0 {
		lines = append(lines, "pods none")
	}
	lines = append(lines, nodeDiagnostics(ctx, client, deploy.Spec.Template.Spec.NodeSelector)...)
	return strings.Join(lines, "\n")
}

func podEvents(ctx context.Context, client kubernetes.Interface, namespace, podName string) []string {
	events, err := client.CoreV1().Events(namespace).List(contextWithoutCancellation(ctx), metav1.ListOptions{})
	if err != nil {
		return []string{fmt.Sprintf("event_list_error pod=%s %v", podName, err)}
	}
	lines := []string{}
	for _, event := range events.Items {
		if event.InvolvedObject.Kind != "Pod" || event.InvolvedObject.Name != podName {
			continue
		}
		lines = append(lines, fmt.Sprintf("event %s %s pod/%s: %s", event.Type, event.Reason, podName, event.Message))
	}
	sort.Strings(lines)
	return lines
}

func contextWithoutCancellation(ctx context.Context) context.Context {
	if ctx.Err() == nil {
		return ctx
	}
	return context.Background()
}

func nodeDiagnostics(ctx context.Context, client kubernetes.Interface, selector map[string]string) []string {
	nodes, err := client.CoreV1().Nodes().List(contextWithoutCancellation(ctx), metav1.ListOptions{})
	if err != nil {
		return []string{fmt.Sprintf("node_list_error %v", err)}
	}
	sort.Slice(nodes.Items, func(i, j int) bool { return nodes.Items[i].Name < nodes.Items[j].Name })
	if len(nodes.Items) == 0 {
		return []string{"nodes none"}
	}
	lines := []string{}
	for _, node := range nodes.Items {
		lines = append(lines, fmt.Sprintf("node name=%s matchesSelector=%t labels=%s", node.Name, nodeMatchesSelector(node.Labels, selector), labels.Set(node.Labels).String()))
	}
	return lines
}

func nodeMatchesSelector(nodeLabels, selector map[string]string) bool {
	for key, want := range selector {
		if nodeLabels[key] != want {
			return false
		}
	}
	return true
}

func deploymentAvailable(deploy *appsv1.Deployment) bool {
	for _, condition := range deploy.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return deploy.Status.AvailableReplicas > 0
}

func resourceName(ref string) string {
	for _, prefix := range []string{"deployment/", "service/", "pvc/"} {
		ref = strings.TrimPrefix(ref, prefix)
	}
	return ref
}

type capacityAdapter struct {
	provider tencentcloud.NodePoolProvider
}

func (a capacityAdapter) EnsureNodePool(ctx context.Context, req fabricruntime.CapacityNodePoolRequest) (fabricruntime.CapacityNodePoolResult, error) {
	result, err := a.provider.EnsureNodePool(ctx, tencentcloud.NodePoolRequest{
		ComputeID:                 req.ComputeID,
		WorkspaceID:               req.WorkspaceID,
		RequestedComputeShapeJSON: req.RequestedComputeShapeJSON,
		ProviderInstanceType:      req.ProviderInstanceType,
	})
	if err != nil {
		return fabricruntime.CapacityNodePoolResult{}, err
	}
	log.Printf("nodepool created compute=%s workspace=%s nodePoolID=%s providerInstanceType=%s", req.ComputeID, req.WorkspaceID, result.NodePoolID, req.ProviderInstanceType)
	return fabricruntime.CapacityNodePoolResult{NodePoolID: result.NodePoolID}, nil
}

func (a capacityAdapter) VerifyNodePool(ctx context.Context, nodePoolID string) (bool, error) {
	verified, err := a.provider.VerifyNodePool(ctx, nodePoolID)
	if err != nil {
		return false, err
	}
	log.Printf("nodepool verify nodePoolID=%s verified=%t", nodePoolID, verified)
	return verified, nil
}

func (a capacityAdapter) DeleteNodePool(ctx context.Context, nodePoolID string) error {
	log.Printf("nodepool delete nodePoolID=%s", nodePoolID)
	return a.provider.DeleteNodePool(ctx, nodePoolID)
}

func boolEnv(key string) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	return err == nil && value
}

func stringEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func int32Env(key string, fallback int32) int32 {
	value := intEnv(key, int(fallback))
	return int32(value)
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
