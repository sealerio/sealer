package image

import (
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	"testing"
)

func TestDefaultImageMetadataService_GetRemoteManifestConfig(t *testing.T) {
	config, err := NewImageMetadataService().GetRemoteImage("registry.cn-qingdao.aliyuncs.com/seadent/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}

	logger.Info(config)
}
