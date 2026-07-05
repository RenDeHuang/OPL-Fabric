package k8s

import (
	"context"
	"errors"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	testingclient "k8s.io/client-go/testing"
)

func TestCreateComputeCreatesDeploymentAndService(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace-image:latest", ImagePullSecretName: "tcr-pull-secret"}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:              "compute-1",
		WorkspaceName:   "Alpha",
		ProductPresetID: "basic",
		CPU:             2,
		MemoryGB:        4,
	})
	if err != nil {
		t.Fatalf("create compute failed: %v", err)
	}
	if !strings.HasPrefix(result.ProviderRef, "deployment/opl-compute-1-") {
		t.Fatalf("provider ref = %s", result.ProviderRef)
	}
	if !strings.HasPrefix(result.ServiceRef, "service/opl-compute-1-") {
		t.Fatalf("service ref = %s", result.ServiceRef)
	}
	name := strings.TrimPrefix(result.ProviderRef, "deployment/")

	deploy, err := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	if deploy.Spec.Template.Spec.Containers[0].Image != "workspace-image:latest" {
		t.Fatalf("image mismatch")
	}
	if deploy.Annotations["oplcloud.cn/compute-id"] != "compute-1" {
		t.Fatalf("raw compute id annotation missing")
	}
	if len(deploy.Spec.Template.Spec.ImagePullSecrets) != 1 || deploy.Spec.Template.Spec.ImagePullSecrets[0].Name != "tcr-pull-secret" {
		t.Fatalf("image pull secrets = %+v, want tcr-pull-secret", deploy.Spec.Template.Spec.ImagePullSecrets)
	}
	if deploy.Spec.Template.Spec.AutomountServiceAccountToken == nil || *deploy.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatalf("automount service account token should be false")
	}
	if deploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != 3000 {
		t.Fatalf("container port mismatch")
	}
	resources := deploy.Spec.Template.Spec.Containers[0].Resources
	if resources.Requests.Cpu().String() != "2" || resources.Limits.Cpu().String() != "2" {
		t.Fatalf("cpu resources mismatch: requests=%s limits=%s", resources.Requests.Cpu(), resources.Limits.Cpu())
	}
	if resources.Requests.Memory().String() != "4Gi" || resources.Limits.Memory().String() != "4Gi" {
		t.Fatalf("memory resources mismatch: requests=%s limits=%s", resources.Requests.Memory(), resources.Limits.Memory())
	}
	if deploy.Spec.Selector.MatchLabels["oplcloud.cn/compute-key"] == "" {
		t.Fatalf("selector missing label-safe compute key")
	}

	service, err := client.CoreV1().Services("opl-fabric").Get(context.Background(), strings.TrimPrefix(result.ServiceRef, "service/"), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("service missing: %v", err)
	}
	if service.Spec.Selector["oplcloud.cn/compute-key"] != deploy.Spec.Template.Labels["oplcloud.cn/compute-key"] {
		t.Fatalf("service selector does not match deployment label")
	}
	if service.Spec.Ports[0].Port != 3000 {
		t.Fatalf("service port mismatch")
	}
}

func TestCreateComputeUsesBoundedDNSNameAndSafeLabels(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace-image:latest"}
	longID := "Compute_" + strings.Repeat("ABC123_", 20)

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:              longID,
		WorkspaceName:   "Alpha",
		ProductPresetID: "basic",
	})
	if err != nil {
		t.Fatalf("create compute failed: %v", err)
	}
	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	if len(name) > 63 {
		t.Fatalf("name length = %d", len(name))
	}
	if strings.Contains(name, "_") {
		t.Fatalf("name contains unsafe character: %s", name)
	}
	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		t.Fatalf("name is not a DNS-1123 label: %v", errs)
	}

	deploy, err := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	if deploy.Annotations["oplcloud.cn/compute-id"] != longID {
		t.Fatalf("raw compute id annotation mismatch")
	}
	for key, value := range deploy.Labels {
		if value == longID {
			t.Fatalf("raw compute id leaked into label %s", key)
		}
		if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
			t.Fatalf("label %s value %q is invalid: %v", key, value, errs)
		}
	}
}

func TestCreateComputeCleansDeploymentWhenServiceCreateFails(t *testing.T) {
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "services", func(action testingclient.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("service_create_failed")
	})
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace-image:latest"}

	_, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:              "compute-1",
		WorkspaceName:   "Alpha",
		ProductPresetID: "basic",
	})
	if err == nil {
		t.Fatal("expected service create failure")
	}
	name := strings.TrimPrefix(k8sName("compute-1"), "deployment/")
	_, getErr := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), name, metav1.GetOptions{})
	if getErr == nil {
		t.Fatal("deployment should be cleaned up after service failure")
	}
}

func TestCreateComputeReportsCleanupFailure(t *testing.T) {
	client := fake.NewSimpleClientset()
	serviceErr := errors.New("service_create_failed")
	deleteErr := errors.New("deployment_delete_failed")
	client.PrependReactor("create", "services", func(action testingclient.Action) (bool, runtime.Object, error) {
		return true, nil, serviceErr
	})
	client.PrependReactor("delete", "deployments", func(action testingclient.Action) (bool, runtime.Object, error) {
		return true, nil, deleteErr
	})
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace-image:latest"}

	_, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:              "compute-1",
		WorkspaceName:   "Alpha",
		ProductPresetID: "basic",
	})
	if !errors.Is(err, serviceErr) {
		t.Fatalf("expected service error, got %v", err)
	}
	if !errors.Is(err, deleteErr) {
		t.Fatalf("expected cleanup error, got %v", err)
	}
}

func TestCreateComputeCarriesCapacityBoundaryMetadata(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace-image:latest"}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:                   "compute-capacity",
		WorkspaceName:        "Capacity",
		ProductPresetID:      "custom",
		ComputeShapeJSON:     `{"cpu":4,"memoryGb":8}`,
		ProviderInstanceType: "SA5.LARGE8",
		CapacityPoolID:       "tencent-cpu-compute-pool",
		IsolationMode:        "workspace_exclusive_cvm",
		NodePoolID:           "np-example",
		RuntimeRef:           "deployment/compute-capacity",
	})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	deploy, err := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	for key, want := range map[string]string{
		"oplcloud.cn/capacity-pool-id":       "tencent-cpu-compute-pool",
		"oplcloud.cn/isolation-mode":         "workspace_exclusive_cvm",
		"oplcloud.cn/node-pool-id":           "np-example",
		"oplcloud.cn/runtime-ref":            "deployment/compute-capacity",
		"oplcloud.cn/provider-instance-type": "SA5.LARGE8",
	} {
		if deploy.Annotations[key] != want {
			t.Fatalf("annotation %s = %q, want %q", key, deploy.Annotations[key], want)
		}
	}
	for key, want := range map[string]string{
		"oplfabric.cn/capacity-model":   "compute-pool",
		"oplfabric.cn/capacity-pool-id": "tencent-cpu-compute-pool",
		"oplfabric.cn/instance-type":    "SA5.LARGE8",
	} {
		if deploy.Spec.Template.Spec.NodeSelector[key] != want {
			t.Fatalf("node selector %s = %q, want %q; selector=%+v", key, deploy.Spec.Template.Spec.NodeSelector[key], want, deploy.Spec.Template.Spec.NodeSelector)
		}
	}

	env := map[string]string{}
	for _, item := range deploy.Spec.Template.Spec.Containers[0].Env {
		env[item.Name] = item.Value
	}
	for key, want := range map[string]string{
		"OPL_PRODUCT_PRESET_ID":  "custom",
		"OPL_COMPUTE_SHAPE_JSON": `{"cpu":4,"memoryGb":8}`,
		"OPL_CAPACITY_POOL_ID":   "tencent-cpu-compute-pool",
		"OPL_ISOLATION_MODE":     "workspace_exclusive_cvm",
	} {
		if env[key] != want {
			t.Fatalf("env %s = %q, want %q", key, env[key], want)
		}
	}
}

func TestCreateComputeInjectsWorkspaceRuntimeConfig(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{
		Client:               client,
		Namespace:            "opl-fabric",
		WorkspaceImage:       "workspace:latest",
		WorkspaceWebUIPort:   3000,
		WorkspaceDataDir:     "/data",
		WorkspaceProjectsDir: "/projects",
		CodexHome:            "/data/codex",
	}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-runtime", WorkspaceName: "Runtime", ProductPresetID: "basic"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	deploy, err := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}

	env := map[string]string{}
	for _, item := range deploy.Spec.Template.Spec.Containers[0].Env {
		env[item.Name] = item.Value
	}
	for key, want := range map[string]string{
		"OPL_PROJECTS_DIR":    "/projects",
		"OPL_WEBUI_AUTH_MODE": "none",
		"OPL_WORKSPACE_ROOT":  "/projects",
		"CODEX_HOME":          "/data/codex",
	} {
		if env[key] != want {
			t.Fatalf("%s = %q, want %q", key, env[key], want)
		}
	}
}

func TestCreateComputeAddsCodexSecretEnvWhenConfigured(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{
		Client:               client,
		Namespace:            "opl-fabric",
		WorkspaceImage:       "workspace:latest",
		WorkspaceWebUIPort:   3000,
		WorkspaceDataDir:     "/data",
		WorkspaceProjectsDir: "/projects",
		CodexHome:            "/data/codex",
		CodexModel:           "gpt-5.5",
		CodexReasoningEffort: "xhigh",
		CodexBaseURL:         "https://gflabtoken.cn/v1",
		CodexAPIKey:          "secret",
		CodexModelProvider:   "gflabtoken",
		CodexProviderName:    "gflabtoken",
	}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-codex", WorkspaceName: "Codex", ProductPresetID: "basic"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	secret, err := client.CoreV1().Secrets("opl-fabric").Get(context.Background(), name+"-env", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("secret missing: %v", err)
	}

	for _, key := range []string{"OPL_CODEX_MODEL", "OPL_CODEX_REASONING_EFFORT", "OPL_CODEX_BASE_URL", "OPL_CODEX_API_KEY"} {
		if len(secret.Data[key]) == 0 {
			t.Fatalf("secret missing %s", key)
		}
	}
}

func TestCreateComputeIsIdempotentAfterPartialSuccess(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{
		Client:         client,
		Namespace:      "opl-fabric",
		WorkspaceImage: "workspace:latest",
		CodexAPIKey:    "secret",
	}

	first, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-retry", WorkspaceName: "Retry", ProductPresetID: "basic"})
	if err != nil {
		t.Fatalf("first CreateCompute: %v", err)
	}
	second, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-retry", WorkspaceName: "Retry", ProductPresetID: "basic"})
	if err != nil {
		t.Fatalf("second CreateCompute: %v", err)
	}
	if second.ProviderRef != first.ProviderRef || second.ServiceRef != first.ServiceRef {
		t.Fatalf("refs = %+v, want %+v", second, first)
	}
}

func TestCreateStorageVolumeCreatesPVC(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", StorageClassName: "cbs"}

	result, err := provider.CreateStorageVolume(context.Background(), CreateStorageVolumeInput{
		ID:     "storage-1",
		SizeGB: 20,
	})
	if err != nil {
		t.Fatalf("CreateStorageVolume: %v", err)
	}

	name := strings.TrimPrefix(result.ProviderRef, "pvc/")
	pvc, err := client.CoreV1().PersistentVolumeClaims("opl-fabric").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("pvc missing: %v", err)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "cbs" {
		t.Fatalf("storage class = %v, want cbs", pvc.Spec.StorageClassName)
	}
	if pvc.Spec.Resources.Requests.Storage().String() != "20Gi" {
		t.Fatalf("storage request = %s, want 20Gi", pvc.Spec.Resources.Requests.Storage())
	}
}

func TestCreateStorageVolumeTreatsExistingPVCAsSuccess(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", StorageClassName: "cbs"}

	first, err := provider.CreateStorageVolume(context.Background(), CreateStorageVolumeInput{ID: "storage-1", SizeGB: 20})
	if err != nil {
		t.Fatalf("first CreateStorageVolume: %v", err)
	}
	second, err := provider.CreateStorageVolume(context.Background(), CreateStorageVolumeInput{ID: "storage-1", SizeGB: 20})
	if err != nil {
		t.Fatalf("second CreateStorageVolume: %v", err)
	}
	if second.ProviderRef != first.ProviderRef {
		t.Fatalf("ProviderRef = %q, want %q", second.ProviderRef, first.ProviderRef)
	}
}

func TestAttachStoragePatchesDeploymentVolumeMount(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest"}
	compute, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-1"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}
	storage, err := provider.CreateStorageVolume(context.Background(), CreateStorageVolumeInput{ID: "storage-1", SizeGB: 10})
	if err != nil {
		t.Fatalf("CreateStorageVolume: %v", err)
	}

	result, err := provider.AttachStorage(context.Background(), AttachStorageInput{
		ID:         "attach-1",
		ComputeRef: compute.ProviderRef,
		StorageRef: storage.ProviderRef,
		MountPath:  "/data",
		SubPath:    "data",
	})
	if err != nil {
		t.Fatalf("AttachStorage: %v", err)
	}

	name := strings.TrimPrefix(compute.ProviderRef, "deployment/")
	deploy, err := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	container := deploy.Spec.Template.Spec.Containers[0]
	if len(container.VolumeMounts) != 1 || container.VolumeMounts[0].MountPath != "/data" {
		t.Fatalf("volume mounts = %+v", container.VolumeMounts)
	}
	if len(deploy.Spec.Template.Spec.Volumes) != 1 || deploy.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName != strings.TrimPrefix(storage.ProviderRef, "pvc/") {
		t.Fatalf("volumes = %+v", deploy.Spec.Template.Spec.Volumes)
	}
	if !strings.HasPrefix(result.ProviderRef, "deployment/") || !strings.Contains(result.ProviderRef, ":pvc/") {
		t.Fatalf("provider ref = %q", result.ProviderRef)
	}
}

func TestCreateWorkspaceEntryCreatesGatewayIngressPath(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceDomain: "workspace.medopl.cn", IngressClassName: "qcloud"}

	err := provider.CreateWorkspaceEntry(context.Background(), CreateWorkspaceEntryInput{
		ID:          "entry-1",
		WorkspaceID: "ws-1",
		Host:        "workspace.medopl.cn",
		Path:        "/w/ws-1/",
		ServiceRef:  "service/opl-compute-1",
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceEntry: %v", err)
	}

	ing, err := client.NetworkingV1().Ingresses("opl-fabric").Get(context.Background(), "opl-fabric-workspace-gateway", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ingress missing: %v", err)
	}
	if ing.Spec.IngressClassName == nil || *ing.Spec.IngressClassName != "qcloud" {
		t.Fatalf("ingress class = %v, want qcloud", ing.Spec.IngressClassName)
	}
	path := ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0]
	if path.Path != "/w/ws-1/" || path.Backend.Service.Name != "opl-compute-1" || path.Backend.Service.Port.Number != 3000 {
		t.Fatalf("ingress path = %+v", path)
	}
}

func TestDetachStorageRemovesDeploymentVolumeMount(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest"}
	compute, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-1"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}
	storage, err := provider.CreateStorageVolume(context.Background(), CreateStorageVolumeInput{ID: "storage-1", SizeGB: 10})
	if err != nil {
		t.Fatalf("CreateStorageVolume: %v", err)
	}
	attach, err := provider.AttachStorage(context.Background(), AttachStorageInput{ID: "attach-1", ComputeRef: compute.ProviderRef, StorageRef: storage.ProviderRef, MountPath: "/data"})
	if err != nil {
		t.Fatalf("AttachStorage: %v", err)
	}

	if err := provider.DetachStorage(context.Background(), DetachStorageInput{ProviderRef: attach.ProviderRef}); err != nil {
		t.Fatalf("DetachStorage: %v", err)
	}

	name := strings.TrimPrefix(compute.ProviderRef, "deployment/")
	deploy, err := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	if len(deploy.Spec.Template.Spec.Containers[0].VolumeMounts) != 0 || len(deploy.Spec.Template.Spec.Volumes) != 0 {
		t.Fatalf("storage still attached: mounts=%+v volumes=%+v", deploy.Spec.Template.Spec.Containers[0].VolumeMounts, deploy.Spec.Template.Spec.Volumes)
	}
}

func TestDestroyComputeDeletesDeploymentServiceAndSecret(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", WorkspaceImage: "workspace:latest", CodexAPIKey: "secret"}
	compute, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-1"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	if err := provider.DestroyCompute(context.Background(), DestroyComputeInput{ProviderRef: compute.ProviderRef, RuntimeRef: compute.ServiceRef}); err != nil {
		t.Fatalf("DestroyCompute: %v", err)
	}

	name := strings.TrimPrefix(compute.ProviderRef, "deployment/")
	if _, err := client.AppsV1().Deployments("opl-fabric").Get(context.Background(), name, metav1.GetOptions{}); err == nil {
		t.Fatal("deployment should be deleted")
	}
	if _, err := client.CoreV1().Services("opl-fabric").Get(context.Background(), name, metav1.GetOptions{}); err == nil {
		t.Fatal("service should be deleted")
	}
	if _, err := client.CoreV1().Secrets("opl-fabric").Get(context.Background(), name+"-env", metav1.GetOptions{}); err == nil {
		t.Fatal("secret should be deleted")
	}
}

func TestDestroyStorageDeletesPVCOnlyWhenRequested(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-fabric", StorageClassName: "cbs"}
	storage, err := provider.CreateStorageVolume(context.Background(), CreateStorageVolumeInput{ID: "storage-1", SizeGB: 10})
	if err != nil {
		t.Fatalf("CreateStorageVolume: %v", err)
	}

	if err := provider.DestroyStorage(context.Background(), DestroyStorageInput{ProviderRef: storage.ProviderRef}); err != nil {
		t.Fatalf("DestroyStorage: %v", err)
	}

	if _, err := client.CoreV1().PersistentVolumeClaims("opl-fabric").Get(context.Background(), strings.TrimPrefix(storage.ProviderRef, "pvc/"), metav1.GetOptions{}); err == nil {
		t.Fatal("pvc should be deleted")
	}
}

var _ = appsv1.Deployment{}
var _ = corev1.Service{}
var _ = networkingv1.Ingress{}
var _ kubernetes.Interface = fake.NewSimpleClientset()
