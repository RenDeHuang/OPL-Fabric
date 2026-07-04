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

func TestResolveNodePoolPlanRequiresProductionInputs(t *testing.T) {
	_, err := ResolveNodePoolPlan(NodePoolResolverConfig{
		Region:          "ap-guangzhou",
		SecretID:        "secret-id",
		SecretKey:       "secret-key",
		MutationAllowed: true,
	})
	if !errors.Is(err, ErrMissingNodePoolConfig) {
		t.Fatalf("error = %v, want %v", err, ErrMissingNodePoolConfig)
	}
}

func TestResolveNodePoolPlanRespectsMutationGate(t *testing.T) {
	plan, err := ResolveNodePoolPlan(NodePoolResolverConfig{
		ClusterID:        "cls-example",
		Region:           "ap-guangzhou",
		SecretID:         "secret-id",
		SecretKey:        "secret-key",
		LaunchConfigJSON: `{"InstanceType":"SA5.LARGE8"}`,
		AutoscalingJSON:  `{"MinSize":0,"MaxSize":3}`,
		MutationAllowed:  false,
	})
	if err != nil {
		t.Fatalf("ResolveNodePoolPlan: %v", err)
	}
	if plan.MutationAllowed {
		t.Fatal("mutation should be blocked by gate")
	}
	if plan.ClusterID != "cls-example" || plan.Region != "ap-guangzhou" {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestResolveNodePoolPlanValidatesJSON(t *testing.T) {
	_, err := ResolveNodePoolPlan(NodePoolResolverConfig{
		ClusterID:        "cls-example",
		Region:           "ap-guangzhou",
		SecretID:         "secret-id",
		SecretKey:        "secret-key",
		LaunchConfigJSON: `{"InstanceType":`,
		AutoscalingJSON:  `{"MinSize":0,"MaxSize":3}`,
		MutationAllowed:  true,
	})
	if !errors.Is(err, ErrInvalidNodePoolJSON) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidNodePoolJSON)
	}
}

func TestResolveNodePoolPlanReturnsTencentSDKBoundary(t *testing.T) {
	plan, err := ResolveNodePoolPlan(NodePoolResolverConfig{
		ClusterID:                 "cls-example",
		Region:                    "ap-guangzhou",
		SecretID:                  "secret-id",
		SecretKey:                 "secret-key",
		LaunchConfigJSON:          `{"InstanceType":"SA5.LARGE8"}`,
		AutoscalingJSON:           `{"MinSize":0,"MaxSize":3}`,
		InstanceChargeType:        "POSTPAID_BY_HOUR",
		DesiredPodNumber:          "0",
		MutationAllowed:           true,
		RequestedComputeShapeJSON: `{"cpu":4,"memoryGb":8}`,
	})
	if err != nil {
		t.Fatalf("ResolveNodePoolPlan: %v", err)
	}
	if !plan.MutationAllowed {
		t.Fatal("mutation should be allowed in plan")
	}
	if plan.SDKAction != "CreateClusterNodePool" {
		t.Fatalf("SDKAction = %q", plan.SDKAction)
	}
	if plan.LaunchConfig["InstanceType"] != "SA5.LARGE8" {
		t.Fatalf("launch config = %+v", plan.LaunchConfig)
	}
	if plan.Autoscaling["MaxSize"] != float64(3) {
		t.Fatalf("autoscaling = %+v", plan.Autoscaling)
	}
}
