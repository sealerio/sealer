// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package image

import (
	"testing"

	"github.com/alibaba/sealer/logger"
)

func Test_Compress(t *testing.T) {

}

func TestDefault_Pull(t *testing.T) {
	err := NewImageService().Pull("seadent/rootfs")
	if err != nil {
		t.Error(err)
	}
}

func TestDefaultImageService_PushWithAnnotations(t *testing.T) {
	err := NewImageService().Push("registry.cn-qingdao.aliyuncs.com/sealer-io/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}

	config, err := NewImageMetadataService().GetRemoteImage("registry.cn-qingdao.aliyuncs.com/sealer-io/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}
	logger.Info(config)
}

func TestDefaultImageService_Delete(t *testing.T) {
	err := NewImageService().Pull("registry.cn-qingdao.aliyuncs.com/seadent/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}
	err = NewImageService().Delete("registry.cn-qingdao.aliyuncs.com/seadent/cloudrootfs:v1.16.9-alpha.5")
	if err != nil {
		t.Error(err)
	}
}
