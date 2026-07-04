package tencentcloud

import (
	"context"
	"errors"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20220501"
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
		SubnetIDs:        "subnet-1",
		SecurityGroupIDs: "sg-1",
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
		SubnetIDs:        "subnet-1",
		SecurityGroupIDs: "sg-1",
		SystemDiskSizeGB: "not-a-number",
		MutationAllowed:  true,
	})
	if !errors.Is(err, ErrMissingNodePoolConfig) {
		t.Fatalf("error = %v, want %v", err, ErrMissingNodePoolConfig)
	}
}

func TestResolveNodePoolPlanReturnsTencentSDKBoundary(t *testing.T) {
	plan, err := ResolveNodePoolPlan(NodePoolResolverConfig{
		ClusterID:                 "cls-example",
		Region:                    "ap-guangzhou",
		SecretID:                  "secret-id",
		SecretKey:                 "secret-key",
		SubnetIDs:                 "subnet-a,subnet-b",
		SecurityGroupIDs:          "sg-a,sg-b",
		SystemDiskType:            "CLOUD_BSSD",
		SystemDiskSizeGB:          "50",
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
	if plan.SDKAction != "CreateNodePool" {
		t.Fatalf("SDKAction = %q", plan.SDKAction)
	}
	if len(plan.SubnetIDs) != 2 || plan.SubnetIDs[0] != "subnet-a" || plan.SubnetIDs[1] != "subnet-b" {
		t.Fatalf("subnet ids = %+v", plan.SubnetIDs)
	}
	if len(plan.SecurityGroupIDs) != 2 || plan.SecurityGroupIDs[0] != "sg-a" || plan.SecurityGroupIDs[1] != "sg-b" {
		t.Fatalf("security group ids = %+v", plan.SecurityGroupIDs)
	}
	if plan.SystemDiskType != "CLOUD_BSSD" || plan.SystemDiskSizeGB != 50 {
		t.Fatalf("system disk = %s/%d", plan.SystemDiskType, plan.SystemDiskSizeGB)
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
			SubnetIDs:        "subnet-1",
			SecurityGroupIDs: "sg-1",
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
			SubnetIDs:          "subnet-1",
			SecurityGroupIDs:   "sg-1",
			SystemDiskType:     "CLOUD_BSSD",
			SystemDiskSizeGB:   "50",
			InstanceChargeType: "POSTPAID_BY_HOUR",
			DesiredPodNumber:   "0",
			MutationAllowed:    true,
		},
	}

	result, err := provider.EnsureNodePool(context.Background(), NodePoolRequest{
		ComputeID:                 "compute-1",
		WorkspaceID:               "ws-1",
		RequestedComputeShapeJSON: `{"cpu":4,"memoryGb":8}`,
		ProviderInstanceType:      "SA5.LARGE8",
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
	if value(client.createRequest.Type) != "Native" {
		t.Fatalf("node pool type = %q", value(client.createRequest.Type))
	}
	if client.createRequest.Native == nil {
		t.Fatal("native config is nil")
	}
	if valueInt64(client.createRequest.Native.Replicas) != 0 {
		t.Fatalf("replicas = %d, want 0", valueInt64(client.createRequest.Native.Replicas))
	}
	if len(client.createRequest.Native.InstanceTypes) != 1 || value(client.createRequest.Native.InstanceTypes[0]) != "SA5.LARGE8" {
		t.Fatalf("instance types = %+v", client.createRequest.Native.InstanceTypes)
	}
	if len(client.createRequest.Native.SubnetIds) != 1 || value(client.createRequest.Native.SubnetIds[0]) != "subnet-1" {
		t.Fatalf("subnet ids = %+v", client.createRequest.Native.SubnetIds)
	}
	if len(client.createRequest.Native.SecurityGroupIds) != 1 || value(client.createRequest.Native.SecurityGroupIds[0]) != "sg-1" {
		t.Fatalf("security groups = %+v", client.createRequest.Native.SecurityGroupIds)
	}
	if client.createRequest.Native.SystemDisk == nil || value(client.createRequest.Native.SystemDisk.DiskType) != "CLOUD_BSSD" || valueInt64(client.createRequest.Native.SystemDisk.DiskSize) != 50 {
		t.Fatalf("system disk = %+v", client.createRequest.Native.SystemDisk)
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
	if client.deleteRequest == nil || value(client.deleteRequest.ClusterId) != "cls-example" || value(client.deleteRequest.NodePoolId) != "np-created" {
		t.Fatalf("delete request = %+v", client.deleteRequest)
	}
}

type fakeTKEClient struct {
	nodePoolID      string
	lifeState       string
	createRequest   *tke.CreateNodePoolRequest
	describeRequest *tke.DescribeNodePoolsRequest
	deleteRequest   *tke.DeleteNodePoolRequest
}

func (c *fakeTKEClient) CreateNodePoolWithContext(_ context.Context, request *tke.CreateNodePoolRequest) (*tke.CreateNodePoolResponse, error) {
	c.createRequest = request
	return &tke.CreateNodePoolResponse{Response: &tke.CreateNodePoolResponseParams{NodePoolId: common.StringPtr(c.nodePoolID)}}, nil
}

func (c *fakeTKEClient) DescribeNodePoolsWithContext(_ context.Context, request *tke.DescribeNodePoolsRequest) (*tke.DescribeNodePoolsResponse, error) {
	c.describeRequest = request
	return &tke.DescribeNodePoolsResponse{Response: &tke.DescribeNodePoolsResponseParams{NodePools: []*tke.NodePool{{NodePoolId: common.StringPtr(c.nodePoolID), LifeState: common.StringPtr(c.lifeState)}}}}, nil
}

func (c *fakeTKEClient) DeleteNodePoolWithContext(_ context.Context, request *tke.DeleteNodePoolRequest) (*tke.DeleteNodePoolResponse, error) {
	c.deleteRequest = request
	return &tke.DeleteNodePoolResponse{Response: &tke.DeleteNodePoolResponseParams{RequestId: common.StringPtr("req-1")}}, nil
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

func valueInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
