package image

import (
	"testing"

	"github.com/alibaba/sealer/logger"
)

func TestDefaultImageMetadataService_GetRemoteManifestConfig(t *testing.T) {
	config, err := NewImageMetadataService().GetRemoteImage("registry.cn-qingdao.aliyuncs.com/seadent/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}

	logger.Info(config)
}
