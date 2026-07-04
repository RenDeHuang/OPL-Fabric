package tencentcloud

import (
	"errors"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
)

var ErrMissingTencentCredential = errors.New("missing_tencent_cloud_credential")

type TKEConfig struct {
	Region    string
	SecretID  string
	SecretKey string
}

func NewTKEClient(cfg TKEConfig) (*tke.Client, error) {
	if cfg.Region == "" || cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, ErrMissingTencentCredential
	}
	credential := common.NewCredential(cfg.SecretID, cfg.SecretKey)
	clientProfile := profile.NewClientProfile()
	return tke.NewClient(credential, cfg.Region, clientProfile)
}
