// Copyright © 2022 Alibaba Group Holding Ltd.
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

package define

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
	osi "github.com/sealerio/sealer/utils/os"
)

type SealerConfig struct {
	RegistryType string `json:"registryType,omitempty"` // docker、oci, default docker, "" is considered to be docker
}

// GetRegistryType read .sealer/config file and get registryType
func GetRegistryType() string {
	sealerConfigFile := common.GetSealerConfigFile()
	sealerConfig := &SealerConfig{}
	if !osi.IsFileExist(sealerConfigFile) {
		logrus.Warn("No .sealer/config exist, registryType consider to be docker")
		return common.DefaultRegistryType
	}
	content, err := os.ReadFile(filepath.Clean(sealerConfigFile))
	if err != nil {
		logrus.Errorf("Read .sealer/config failed:%v, registryType consider to be docker", err)
		return common.DefaultRegistryType
	}
	err = json.Unmarshal(content, sealerConfig)
	if err != nil {
		logrus.Errorf("unmarshal .sealer/config failed:%v, registryType consider to be docker", err)
		return common.DefaultRegistryType
	}
	if sealerConfig.RegistryType != common.OCIRegistryType {
		logrus.Errorf("unsupport registryType:%v, registryType consider to be docker", sealerConfig.RegistryType)
		return common.DefaultRegistryType
	}
	logrus.Printf("registry type: %v", sealerConfig.RegistryType)
	return sealerConfig.RegistryType
}
