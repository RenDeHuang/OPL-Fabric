package tencentcloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20220501"
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
	SubnetIDs                 string
	SecurityGroupIDs          string
	SystemDiskType            string
	SystemDiskSizeGB          string
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
	SubnetIDs                 []string
	SecurityGroupIDs          []string
	SystemDiskType            string
	SystemDiskSizeGB          int64
	RequestedComputeShapeJSON string
}

type TKEAPI interface {
	CreateNodePoolWithContext(context.Context, *tke.CreateNodePoolRequest) (*tke.CreateNodePoolResponse, error)
	DescribeNodePoolsWithContext(context.Context, *tke.DescribeNodePoolsRequest) (*tke.DescribeNodePoolsResponse, error)
	DeleteNodePoolWithContext(context.Context, *tke.DeleteNodePoolRequest) (*tke.DeleteNodePoolResponse, error)
}

type NodePoolProvider struct {
	Client TKEAPI
	Config NodePoolResolverConfig
}

type NodePoolRequest struct {
	ComputeAllocationID       string
	WorkspaceID               string
	RequestedComputeShapeJSON string
	ProviderInstanceType      string
	CapacityPoolID            string
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
	clientProfile.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	return tke.NewClient(credential, cfg.Region, clientProfile)
}

func ResolveNodePoolPlan(cfg NodePoolResolverConfig) (NodePoolPlan, error) {
	subnetIDs := splitCSV(cfg.SubnetIDs)
	securityGroupIDs := splitCSV(cfg.SecurityGroupIDs)
	if cfg.ClusterID == "" || cfg.Region == "" || cfg.SecretID == "" || cfg.SecretKey == "" || len(subnetIDs) == 0 || len(securityGroupIDs) == 0 {
		return NodePoolPlan{}, ErrMissingNodePoolConfig
	}
	desiredPodNumber := int64(0)
	if cfg.DesiredPodNumber != "" {
		parsed, err := strconv.ParseInt(cfg.DesiredPodNumber, 10, 64)
		if err != nil {
			return NodePoolPlan{}, ErrMissingNodePoolConfig
		}
		desiredPodNumber = parsed
	}
	systemDiskSizeGB := int64(50)
	if cfg.SystemDiskSizeGB != "" {
		parsed, err := strconv.ParseInt(cfg.SystemDiskSizeGB, 10, 64)
		if err != nil || parsed <= 0 {
			return NodePoolPlan{}, ErrMissingNodePoolConfig
		}
		systemDiskSizeGB = parsed
	}
	return NodePoolPlan{
		SDKAction:                 "CreateNodePool",
		ClusterID:                 cfg.ClusterID,
		Region:                    cfg.Region,
		InstanceChargeType:        defaultString(cfg.InstanceChargeType, "POSTPAID_BY_HOUR"),
		DesiredPodNumber:          desiredPodNumber,
		MutationAllowed:           cfg.MutationAllowed,
		SubnetIDs:                 subnetIDs,
		SecurityGroupIDs:          securityGroupIDs,
		SystemDiskType:            defaultString(cfg.SystemDiskType, "CLOUD_BSSD"),
		SystemDiskSizeGB:          systemDiskSizeGB,
		RequestedComputeShapeJSON: cfg.RequestedComputeShapeJSON,
	}, nil
}

func (p NodePoolProvider) EnsureNodePool(ctx context.Context, req NodePoolRequest) (NodePoolResult, error) {
	plan, err := ResolveNodePoolPlan(NodePoolResolverConfig{
		ClusterID:                 p.Config.ClusterID,
		Region:                    p.Config.Region,
		SecretID:                  p.Config.SecretID,
		SecretKey:                 p.Config.SecretKey,
		SubnetIDs:                 p.Config.SubnetIDs,
		SecurityGroupIDs:          p.Config.SecurityGroupIDs,
		SystemDiskType:            p.Config.SystemDiskType,
		SystemDiskSizeGB:          p.Config.SystemDiskSizeGB,
		InstanceChargeType:        p.Config.InstanceChargeType,
		DesiredPodNumber:          p.Config.DesiredPodNumber,
		MutationAllowed:           p.Config.MutationAllowed,
		RequestedComputeShapeJSON: req.RequestedComputeShapeJSON,
	})
	if err != nil {
		return NodePoolResult{}, err
	}
	if strings.TrimSpace(req.ProviderInstanceType) == "" {
		return NodePoolResult{}, ErrMissingNodePoolConfig
	}
	poolName := computePoolName(req)
	client, err := p.client()
	if err != nil {
		return NodePoolResult{}, err
	}
	existingID, err := p.findNodePoolByName(ctx, client, poolName)
	if err != nil {
		return NodePoolResult{}, err
	}
	if existingID != "" {
		return NodePoolResult{NodePoolID: existingID}, nil
	}
	if !p.Config.MutationAllowed {
		return NodePoolResult{}, ErrNodePoolMutationNotAllowed
	}
	request := tke.NewCreateNodePoolRequest()
	request.ClusterId = common.StringPtr(plan.ClusterID)
	request.Name = common.StringPtr(poolName)
	request.Type = common.StringPtr("Native")
	request.DeletionProtection = common.BoolPtr(false)
	request.Labels = []*tke.Label{
		{Name: common.StringPtr("oplfabric.cn/capacity-model"), Value: common.StringPtr("compute-pool")},
		{Name: common.StringPtr("oplfabric.cn/capacity-pool-id"), Value: common.StringPtr(req.CapacityPoolID)},
		{Name: common.StringPtr("oplfabric.cn/instance-type"), Value: common.StringPtr(req.ProviderInstanceType)},
	}
	request.Native = &tke.CreateNativeNodePoolParam{
		Scaling: &tke.MachineSetScaling{
			MinReplicas:  common.Int64Ptr(0),
			MaxReplicas:  common.Int64Ptr(1),
			CreatePolicy: common.StringPtr("ZonePriority"),
		},
		SubnetIds:          stringsToPtrs(plan.SubnetIDs),
		InstanceChargeType: common.StringPtr(plan.InstanceChargeType),
		SystemDisk: &tke.Disk{
			DiskType: common.StringPtr(plan.SystemDiskType),
			DiskSize: common.Int64Ptr(plan.SystemDiskSizeGB),
		},
		InstanceTypes:      []*string{common.StringPtr(req.ProviderInstanceType)},
		SecurityGroupIds:   stringsToPtrs(plan.SecurityGroupIDs),
		AutoRepair:         common.BoolPtr(true),
		EnableAutoscaling:  common.BoolPtr(true),
		Replicas:           common.Int64Ptr(plan.DesiredPodNumber),
		InternetAccessible: &tke.InternetAccessible{MaxBandwidthOut: common.Int64Ptr(0), ChargeType: common.StringPtr("TRAFFIC_POSTPAID_BY_HOUR")},
		MachineType:        common.StringPtr("Native"),
		AutomationService:  common.BoolPtr(true),
		RuntimeRootDir:     common.StringPtr("/var/lib/containerd"),
	}
	response, err := client.CreateNodePoolWithContext(ctx, request)
	if err != nil {
		return NodePoolResult{}, err
	}
	if response == nil || response.Response == nil || response.Response.NodePoolId == nil || *response.Response.NodePoolId == "" {
		return NodePoolResult{}, ErrMissingNodePoolConfig
	}
	return NodePoolResult{NodePoolID: *response.Response.NodePoolId}, nil
}

func (p NodePoolProvider) findNodePoolByName(ctx context.Context, client TKEAPI, name string) (string, error) {
	request := tke.NewDescribeNodePoolsRequest()
	request.ClusterId = common.StringPtr(p.Config.ClusterID)
	request.Limit = common.Int64Ptr(100)
	request.Filters = []*tke.Filter{{
		Name:   common.StringPtr("NodePoolNames"),
		Values: []*string{common.StringPtr(name)},
	}}
	response, err := client.DescribeNodePoolsWithContext(ctx, request)
	if err != nil {
		return "", err
	}
	if response == nil || response.Response == nil {
		return "", nil
	}
	for _, pool := range response.Response.NodePools {
		if pool == nil || pool.NodePoolId == nil || *pool.NodePoolId == "" {
			continue
		}
		if pool.Name != nil && *pool.Name != name {
			continue
		}
		return *pool.NodePoolId, nil
	}
	return "", nil
}

func (p NodePoolProvider) VerifyNodePool(ctx context.Context, nodePoolID string) (bool, error) {
	if nodePoolID == "" {
		return false, ErrMissingNodePoolConfig
	}
	client, err := p.client()
	if err != nil {
		return false, err
	}
	request := tke.NewDescribeNodePoolsRequest()
	request.ClusterId = common.StringPtr(p.Config.ClusterID)
	request.Limit = common.Int64Ptr(100)
	request.Filters = []*tke.Filter{{
		Name:   common.StringPtr("NodePoolsId"),
		Values: []*string{common.StringPtr(nodePoolID)},
	}}
	response, err := client.DescribeNodePoolsWithContext(ctx, request)
	if err != nil {
		return false, err
	}
	if response == nil || response.Response == nil {
		return false, nil
	}
	for _, pool := range response.Response.NodePools {
		if pool == nil || pool.NodePoolId == nil || *pool.NodePoolId != nodePoolID {
			continue
		}
		if pool.LifeState == nil {
			return true, nil
		}
		state := strings.ToLower(*pool.LifeState)
		return state == "normal" || state == "creating" || state == "running", nil
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
	request := tke.NewDeleteNodePoolRequest()
	request.ClusterId = common.StringPtr(p.Config.ClusterID)
	request.NodePoolId = common.StringPtr(nodePoolID)
	_, err = client.DeleteNodePoolWithContext(ctx, request)
	return err
}

func (p NodePoolProvider) client() (TKEAPI, error) {
	if p.Client != nil {
		return p.Client, nil
	}
	return NewTKEClient(TKEConfig{Region: p.Config.Region, SecretID: p.Config.SecretID, SecretKey: p.Config.SecretKey})
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func computePoolName(req NodePoolRequest) string {
	key := strings.TrimSpace(req.ProviderInstanceType)
	if key == "" {
		key = strings.TrimSpace(req.CapacityPoolID)
	}
	if key == "" {
		key = shortHash(req.RequestedComputeShapeJSON)
	}
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return '-'
	}, key)
	clean = strings.Trim(strings.Join(strings.FieldsFunc(clean, func(r rune) bool { return r == '-' }), "-"), "-")
	if clean == "" {
		clean = "workspace"
	}
	name := "opl-pool-" + clean
	if len(name) > 60 {
		name = name[:51] + "-" + shortHash(key)
	}
	return name
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func stringsToPtrs(values []string) []*string {
	result := make([]*string, 0, len(values))
	for _, value := range values {
		result = append(result, common.StringPtr(value))
	}
	return result
}
