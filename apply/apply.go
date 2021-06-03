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

package apply

import (
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type Interface interface {
	Apply() error
	Delete() error
}

func NewApplierFromFile(clusterfile string) Interface {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterfile, cluster); err != nil {
		logger.Error("apply cloud cluster failed", err)
		return nil
	}
	return NewApplier(cluster)
}

func NewApplier(cluster *v1.Cluster) Interface {
	switch cluster.Spec.Provider {
	case common.AliCloud:
		return NewAliCloudProvider(cluster)
	}
	return NewDefaultApplier(cluster)
}

func saveClusterfile(cluster *v1.Cluster) error {
	fileName := common.GetClusterWorkClusterfile(cluster.Name)
	err := utils.MkFileFullPathDir(fileName)
	if err != nil {
		return fmt.Errorf("mkdir failed %s %v", fileName, err)
	}
	err = utils.MarshalYamlToFile(fileName, cluster)
	if err != nil {
		return fmt.Errorf("marshal cluster file failed %v", err)
	}
	return nil
}
