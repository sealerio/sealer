package image

import (
	"encoding/json"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	"io/ioutil"
	"testing"
)

func Test_Compress(t *testing.T) {

}

func Test_Unmarshal(t *testing.T) {
	const imageMapJson = `
	{
	 "kuberentes:v1.18.8": {"name":"kuberentes:v1.18.8","id":"f7de07561dba","driver":"overlay2"},
	 "kuberentes:v1.18.9": {"name":"kuberentes:v1.18.9","id":"f7de07561dba","driver":"default"}
	}
	`
	err := ioutil.WriteFile("/tmp/image-list-test.json", []byte(imageMapJson), 0644)
	if err != nil {
		t.Error(err)
	}

	images, err := imagesMap("/tmp/image-list-test.json")
	if err != nil {
		t.Error(err)
	}

	res, err := json.MarshalIndent(images, "", DefaultJsonIndent)
	err = ioutil.WriteFile("/tmp/image-list-test.json", res, 0644)
	if err != nil {
		t.Error(err)
	}
}

func TestDefault_Pull(t *testing.T) {
	err := NewImageService().Pull("registry.cn-qingdao.aliyuncs.com/seadent/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}
}

func TestDefaultImageService_PushWithAnnotations(t *testing.T) {
	err := NewImageService().Push("registry.cn-qingdao.aliyuncs.com/seadent/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}

	config, err := NewImageMetadataService().GetRemoteImage("registry.cn-qingdao.aliyuncs.com/seadent/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}
	logger.Info(config)
}
