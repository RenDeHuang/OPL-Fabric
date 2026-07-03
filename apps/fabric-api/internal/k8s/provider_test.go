package k8s

import (
	"context"
	"errors"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	testingclient "k8s.io/client-go/testing"
)

func TestCreateComputeCreatesDeploymentAndService(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            "compute-1",
		WorkspaceName: "Alpha",
		PackageID:     "basic",
		CPU:           2,
		MemoryGB:      4,
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

	deploy, err := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment missing: %v", err)
	}
	if deploy.Spec.Template.Spec.Containers[0].Image != "workspace-image:latest" {
		t.Fatalf("image mismatch")
	}
	if deploy.Annotations["oplcloud.cn/compute-id"] != "compute-1" {
		t.Fatalf("raw compute id annotation missing")
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

	service, err := client.CoreV1().Services("opl-cloud").Get(context.Background(), strings.TrimPrefix(result.ServiceRef, "service/"), metav1.GetOptions{})
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
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}
	longID := "Compute_" + strings.Repeat("ABC123_", 20)

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            longID,
		WorkspaceName: "Alpha",
		PackageID:     "basic",
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

	deploy, err := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), name, metav1.GetOptions{})
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
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}

	_, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            "compute-1",
		WorkspaceName: "Alpha",
		PackageID:     "basic",
	})
	if err == nil {
		t.Fatal("expected service create failure")
	}
	name := strings.TrimPrefix(k8sName("compute-1"), "deployment/")
	_, getErr := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), name, metav1.GetOptions{})
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
	provider := Provider{Client: client, Namespace: "opl-cloud", WorkspaceImage: "workspace-image:latest"}

	_, err := provider.CreateCompute(context.Background(), CreateComputeInput{
		ID:            "compute-1",
		WorkspaceName: "Alpha",
		PackageID:     "basic",
	})
	if !errors.Is(err, serviceErr) {
		t.Fatalf("expected service error, got %v", err)
	}
	if !errors.Is(err, deleteErr) {
		t.Fatalf("expected cleanup error, got %v", err)
	}
}

func TestCreateComputeInjectsWorkspaceRuntimeConfig(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := Provider{
		Client:               client,
		Namespace:            "opl-cloud",
		WorkspaceImage:       "workspace:latest",
		WorkspaceWebUIPort:   3000,
		WorkspaceDataDir:     "/data",
		WorkspaceProjectsDir: "/projects",
		CodexHome:            "/data/codex",
	}

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-runtime", WorkspaceName: "Runtime", PackageID: "basic"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	deploy, err := client.AppsV1().Deployments("opl-cloud").Get(context.Background(), name, metav1.GetOptions{})
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
		Namespace:            "opl-cloud",
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

	result, err := provider.CreateCompute(context.Background(), CreateComputeInput{ID: "compute-codex", WorkspaceName: "Codex", PackageID: "basic"})
	if err != nil {
		t.Fatalf("CreateCompute: %v", err)
	}

	name := strings.TrimPrefix(result.ProviderRef, "deployment/")
	secret, err := client.CoreV1().Secrets("opl-cloud").Get(context.Background(), name+"-env", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("secret missing: %v", err)
	}

	for _, key := range []string{"OPL_CODEX_MODEL", "OPL_CODEX_REASONING_EFFORT", "OPL_CODEX_BASE_URL", "OPL_CODEX_API_KEY"} {
		if len(secret.Data[key]) == 0 {
			t.Fatalf("secret missing %s", key)
		}
	}
}

var _ = appsv1.Deployment{}
var _ = corev1.Service{}
var _ kubernetes.Interface = fake.NewSimpleClientset()
