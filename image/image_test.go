package image

import (
	"testing"

	"github.com/alibaba/sealer/logger"
)

func Test_Compress(t *testing.T) {

}

func TestDefault_Pull(t *testing.T) {
	err := NewImageService().Pull("registry.cn-qingdao.aliyuncs.com/sealer/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}
}

func TestDefaultImageService_PushWithAnnotations(t *testing.T) {
	err := NewImageService().Push("registry.cn-qingdao.aliyuncs.com/sealer/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}

	config, err := NewImageMetadataService().GetRemoteImage("registry.cn-qingdao.aliyuncs.com/sealer/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}
	logger.Info(config)
}
