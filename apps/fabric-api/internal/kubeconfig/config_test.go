package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRESTConfigUsesConfiguredKubeconfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1:6443
    insecure-skip-tls-verify: true
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: test-token
`), 0600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	restConfig, err := LoadRESTConfig(path)
	if err != nil {
		t.Fatalf("LoadRESTConfig: %v", err)
	}
	if restConfig.Host != "https://127.0.0.1:6443" {
		t.Fatalf("Host = %q", restConfig.Host)
	}
	if restConfig.BearerToken != "test-token" {
		t.Fatalf("BearerToken not loaded")
	}
}
