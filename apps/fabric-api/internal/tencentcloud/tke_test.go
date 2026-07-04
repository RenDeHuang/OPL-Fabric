package tencentcloud

import (
	"context"
	"errors"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
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

func TestNodePoolProviderRequiresMutationGateForCreateAndDelete(t *testing.T) {
	provider := NodePoolProvider{
		Client: &fakeTKEClient{},
		Config: NodePoolResolverConfig{
			ClusterID:        "cls-example",
			Region:           "ap-guangzhou",
			SecretID:         "secret-id",
			SecretKey:        "secret-key",
			LaunchConfigJSON: `{"InstanceType":"SA5.LARGE8"}`,
			AutoscalingJSON:  `{"MinSize":0,"MaxSize":3}`,
			MutationAllowed:  false,
		},
	}

	_, err := provider.EnsureNodePool(context.Background(), NodePoolRequest{ComputeID: "compute-1"})
	if !errors.Is(err, ErrNodePoolMutationNotAllowed) {
		t.Fatalf("EnsureNodePool error = %v, want %v", err, ErrNodePoolMutationNotAllowed)
	}
	if err := provider.DeleteNodePool(context.Background(), "np-1"); !errors.Is(err, ErrNodePoolMutationNotAllowed) {
		t.Fatalf("DeleteNodePool error = %v, want %v", err, ErrNodePoolMutationNotAllowed)
	}
}

func TestNodePoolProviderCreatesVerifiesAndDeletesNodePool(t *testing.T) {
	client := &fakeTKEClient{nodePoolID: "np-created", lifeState: "normal"}
	provider := NodePoolProvider{
		Client: client,
		Config: NodePoolResolverConfig{
			ClusterID:          "cls-example",
			Region:             "ap-guangzhou",
			SecretID:           "secret-id",
			SecretKey:          "secret-key",
			LaunchConfigJSON:   `{"InstanceType":"SA5.LARGE8"}`,
			AutoscalingJSON:    `{"MinSize":0,"MaxSize":3}`,
			InstanceChargeType: "POSTPAID_BY_HOUR",
			DesiredPodNumber:   "0",
			MutationAllowed:    true,
		},
	}

	result, err := provider.EnsureNodePool(context.Background(), NodePoolRequest{
		ComputeID:                 "compute-1",
		WorkspaceID:               "ws-1",
		RequestedComputeShapeJSON: `{"cpu":4,"memoryGb":8}`,
	})
	if err != nil {
		t.Fatalf("EnsureNodePool: %v", err)
	}
	if result.NodePoolID != "np-created" {
		t.Fatalf("NodePoolID = %q", result.NodePoolID)
	}
	if client.createRequest == nil || value(client.createRequest.ClusterId) != "cls-example" || value(client.createRequest.Name) != "opl-compute-1" {
		t.Fatalf("create request = %+v", client.createRequest)
	}
	if value(client.createRequest.AutoScalingGroupPara) != `{"MinSize":0,"MaxSize":3}` {
		t.Fatalf("autoscaling = %q", value(client.createRequest.AutoScalingGroupPara))
	}
	if value(client.createRequest.LaunchConfigurePara) != `{"InstanceType":"SA5.LARGE8"}` {
		t.Fatalf("launch = %q", value(client.createRequest.LaunchConfigurePara))
	}

	verified, err := provider.VerifyNodePool(context.Background(), "np-created")
	if err != nil {
		t.Fatalf("VerifyNodePool: %v", err)
	}
	if !verified {
		t.Fatal("verified = false, want true")
	}
	if client.describeRequest == nil || value(client.describeRequest.ClusterId) != "cls-example" {
		t.Fatalf("describe request = %+v", client.describeRequest)
	}

	if err := provider.DeleteNodePool(context.Background(), "np-created"); err != nil {
		t.Fatalf("DeleteNodePool: %v", err)
	}
	if client.deleteRequest == nil || value(client.deleteRequest.ClusterId) != "cls-example" || len(client.deleteRequest.NodePoolIds) != 1 || value(client.deleteRequest.NodePoolIds[0]) != "np-created" {
		t.Fatalf("delete request = %+v", client.deleteRequest)
	}
	if valueBool(client.deleteRequest.KeepInstance) {
		t.Fatal("KeepInstance = true, want false")
	}
}

type fakeTKEClient struct {
	nodePoolID      string
	lifeState       string
	createRequest   *tke.CreateClusterNodePoolRequest
	describeRequest *tke.DescribeClusterNodePoolsRequest
	deleteRequest   *tke.DeleteClusterNodePoolRequest
}

func (c *fakeTKEClient) CreateClusterNodePoolWithContext(_ context.Context, request *tke.CreateClusterNodePoolRequest) (*tke.CreateClusterNodePoolResponse, error) {
	c.createRequest = request
	return &tke.CreateClusterNodePoolResponse{Response: &tke.CreateClusterNodePoolResponseParams{NodePoolId: common.StringPtr(c.nodePoolID)}}, nil
}

func (c *fakeTKEClient) DescribeClusterNodePoolsWithContext(_ context.Context, request *tke.DescribeClusterNodePoolsRequest) (*tke.DescribeClusterNodePoolsResponse, error) {
	c.describeRequest = request
	return &tke.DescribeClusterNodePoolsResponse{Response: &tke.DescribeClusterNodePoolsResponseParams{NodePoolSet: []*tke.NodePool{{NodePoolId: common.StringPtr(c.nodePoolID), LifeState: common.StringPtr(c.lifeState)}}}}, nil
}

func (c *fakeTKEClient) DeleteClusterNodePoolWithContext(_ context.Context, request *tke.DeleteClusterNodePoolRequest) (*tke.DeleteClusterNodePoolResponse, error) {
	c.deleteRequest = request
	return &tke.DeleteClusterNodePoolResponse{Response: &tke.DeleteClusterNodePoolResponseParams{RequestId: common.StringPtr("req-1")}}, nil
}

func value(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func valueBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}
