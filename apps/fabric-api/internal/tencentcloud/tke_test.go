package tencentcloud

import (
	"errors"
	"testing"
)

func TestNewTKEClientRequiresCredentialBoundary(t *testing.T) {
	_, err := NewTKEClient(TKEConfig{Region: "ap-guangzhou", SecretID: "secret-id"})
	if !errors.Is(err, ErrMissingTencentCredential) {
		t.Fatalf("error = %v, want %v", err, ErrMissingTencentCredential)
	}
}

func TestNewTKEClientBuildsTencentSDKClient(t *testing.T) {
	client, err := NewTKEClient(TKEConfig{
		Region:    "ap-guangzhou",
		SecretID:  "secret-id",
		SecretKey: "secret-key",
	})
	if err != nil {
		t.Fatalf("NewTKEClient: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
}
