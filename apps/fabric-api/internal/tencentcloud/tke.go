package tencentcloud

import (
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
