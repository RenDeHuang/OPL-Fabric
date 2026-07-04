package tencentcloud

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
)

var ErrMissingTencentCredential = errors.New("missing_tencent_cloud_credential")
var ErrMissingNodePoolConfig = errors.New("missing_nodepool_config")
var ErrInvalidNodePoolJSON = errors.New("invalid_nodepool_json")
var ErrNodePoolMutationNotAllowed = errors.New("nodepool_mutation_not_allowed")

type TKEConfig struct {
	Region    string
	SecretID  string
	SecretKey string
}

type NodePoolResolverConfig struct {
	ClusterID                 string
	Region                    string
	SecretID                  string
	SecretKey                 string
	LaunchConfigJSON          string
	AutoscalingJSON           string
	InstanceChargeType        string
	DesiredPodNumber          string
	MutationAllowed           bool
	RequestedComputeShapeJSON string
}

type NodePoolPlan struct {
	SDKAction                 string
	ClusterID                 string
	Region                    string
	InstanceChargeType        string
	DesiredPodNumber          int64
	MutationAllowed           bool
	LaunchConfig              map[string]any
	Autoscaling               map[string]any
	RequestedComputeShapeJSON string
}

type TKEAPI interface {
	CreateClusterNodePoolWithContext(context.Context, *tke.CreateClusterNodePoolRequest) (*tke.CreateClusterNodePoolResponse, error)
	DescribeClusterNodePoolsWithContext(context.Context, *tke.DescribeClusterNodePoolsRequest) (*tke.DescribeClusterNodePoolsResponse, error)
	DeleteClusterNodePoolWithContext(context.Context, *tke.DeleteClusterNodePoolRequest) (*tke.DeleteClusterNodePoolResponse, error)
}

type NodePoolProvider struct {
	Client TKEAPI
	Config NodePoolResolverConfig
}

type NodePoolRequest struct {
	ComputeID                 string
	WorkspaceID               string
	RequestedComputeShapeJSON string
}

type NodePoolResult struct {
	NodePoolID string
}

func NewTKEClient(cfg TKEConfig) (*tke.Client, error) {
	if cfg.Region == "" || cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, ErrMissingTencentCredential
	}
	credential := common.NewCredential(cfg.SecretID, cfg.SecretKey)
	clientProfile := profile.NewClientProfile()
	return tke.NewClient(credential, cfg.Region, clientProfile)
}

func ResolveNodePoolPlan(cfg NodePoolResolverConfig) (NodePoolPlan, error) {
	if cfg.ClusterID == "" || cfg.Region == "" || cfg.SecretID == "" || cfg.SecretKey == "" || cfg.LaunchConfigJSON == "" || cfg.AutoscalingJSON == "" {
		return NodePoolPlan{}, ErrMissingNodePoolConfig
	}
	launchConfig, err := parseJSONMap(cfg.LaunchConfigJSON)
	if err != nil {
		return NodePoolPlan{}, err
	}
	autoscaling, err := parseJSONMap(cfg.AutoscalingJSON)
	if err != nil {
		return NodePoolPlan{}, err
	}
	desiredPodNumber := int64(0)
	if cfg.DesiredPodNumber != "" {
		parsed, err := strconv.ParseInt(cfg.DesiredPodNumber, 10, 64)
		if err != nil {
			return NodePoolPlan{}, ErrMissingNodePoolConfig
		}
		desiredPodNumber = parsed
	}
	return NodePoolPlan{
		SDKAction:                 "CreateClusterNodePool",
		ClusterID:                 cfg.ClusterID,
		Region:                    cfg.Region,
		InstanceChargeType:        defaultString(cfg.InstanceChargeType, "POSTPAID_BY_HOUR"),
		DesiredPodNumber:          desiredPodNumber,
		MutationAllowed:           cfg.MutationAllowed,
		LaunchConfig:              launchConfig,
		Autoscaling:               autoscaling,
		RequestedComputeShapeJSON: cfg.RequestedComputeShapeJSON,
	}, nil
}

func (p NodePoolProvider) EnsureNodePool(ctx context.Context, req NodePoolRequest) (NodePoolResult, error) {
	if !p.Config.MutationAllowed {
		return NodePoolResult{}, ErrNodePoolMutationNotAllowed
	}
	plan, err := ResolveNodePoolPlan(NodePoolResolverConfig{
		ClusterID:                 p.Config.ClusterID,
		Region:                    p.Config.Region,
		SecretID:                  p.Config.SecretID,
		SecretKey:                 p.Config.SecretKey,
		LaunchConfigJSON:          p.Config.LaunchConfigJSON,
		AutoscalingJSON:           p.Config.AutoscalingJSON,
		InstanceChargeType:        p.Config.InstanceChargeType,
		DesiredPodNumber:          p.Config.DesiredPodNumber,
		MutationAllowed:           p.Config.MutationAllowed,
		RequestedComputeShapeJSON: req.RequestedComputeShapeJSON,
	})
	if err != nil {
		return NodePoolResult{}, err
	}
	client, err := p.client()
	if err != nil {
		return NodePoolResult{}, err
	}
	request := tke.NewCreateClusterNodePoolRequest()
	request.ClusterId = common.StringPtr(plan.ClusterID)
	request.AutoScalingGroupPara = common.StringPtr(p.Config.AutoscalingJSON)
	request.LaunchConfigurePara = common.StringPtr(p.Config.LaunchConfigJSON)
	request.EnableAutoscale = common.BoolPtr(true)
	request.Name = common.StringPtr(nodePoolName(req.ComputeID))
	request.DeletionProtection = common.BoolPtr(false)
	response, err := client.CreateClusterNodePoolWithContext(ctx, request)
	if err != nil {
		return NodePoolResult{}, err
	}
	if response == nil || response.Response == nil || response.Response.NodePoolId == nil || *response.Response.NodePoolId == "" {
		return NodePoolResult{}, ErrMissingNodePoolConfig
	}
	return NodePoolResult{NodePoolID: *response.Response.NodePoolId}, nil
}

func (p NodePoolProvider) VerifyNodePool(ctx context.Context, nodePoolID string) (bool, error) {
	if nodePoolID == "" {
		return false, ErrMissingNodePoolConfig
	}
	client, err := p.client()
	if err != nil {
		return false, err
	}
	request := tke.NewDescribeClusterNodePoolsRequest()
	request.ClusterId = common.StringPtr(p.Config.ClusterID)
	request.Filters = []*tke.Filter{{
		Name:   common.StringPtr("NodePoolsId"),
		Values: []*string{common.StringPtr(nodePoolID)},
	}}
	response, err := client.DescribeClusterNodePoolsWithContext(ctx, request)
	if err != nil {
		return false, err
	}
	if response == nil || response.Response == nil {
		return false, nil
	}
	for _, pool := range response.Response.NodePoolSet {
		if pool == nil || pool.NodePoolId == nil || *pool.NodePoolId != nodePoolID {
			continue
		}
		if pool.LifeState == nil {
			return true, nil
		}
		return *pool.LifeState == "normal" || *pool.LifeState == "creating", nil
	}
	return false, nil
}

func (p NodePoolProvider) DeleteNodePool(ctx context.Context, nodePoolID string) error {
	if !p.Config.MutationAllowed {
		return ErrNodePoolMutationNotAllowed
	}
	if nodePoolID == "" {
		return ErrMissingNodePoolConfig
	}
	client, err := p.client()
	if err != nil {
		return err
	}
	request := tke.NewDeleteClusterNodePoolRequest()
	request.ClusterId = common.StringPtr(p.Config.ClusterID)
	request.NodePoolIds = []*string{common.StringPtr(nodePoolID)}
	request.KeepInstance = common.BoolPtr(false)
	_, err = client.DeleteClusterNodePoolWithContext(ctx, request)
	return err
}

func (p NodePoolProvider) client() (TKEAPI, error) {
	if p.Client != nil {
		return p.Client, nil
	}
	return NewTKEClient(TKEConfig{Region: p.Config.Region, SecretID: p.Config.SecretID, SecretKey: p.Config.SecretKey})
}

func parseJSONMap(value string) (map[string]any, error) {
	result := map[string]any{}
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, ErrInvalidNodePoolJSON
	}
	return result, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func nodePoolName(computeID string) string {
	if computeID == "" {
		return "opl-workspace"
	}
	return "opl-" + computeID
}
