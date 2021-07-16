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
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type Interface interface {
	Apply() error
	Delete() error
}

func NewApplierFromFile(clusterfile string) (Interface, error) {
	cluster, err := GetClusterFromFile(clusterfile)
	if err != nil {
		return nil, err
	}
	return NewApplier(cluster)
}

func GetClusterFromFile(filepath string) (cluster *v1.Cluster, err error) {
	cluster = &v1.Cluster{}
	if err = utils.UnmarshalYamlFile(filepath, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster from %s, %v", filepath, err)
	}
	cluster.SetAnnotations(common.ClusterfileName, filepath)
	return cluster, nil
}

func NewApplier(cluster *v1.Cluster) (Interface, error) {
	switch cluster.Spec.Provider {
	case common.AliCloud:
		return NewAliCloudProvider(cluster)
	case common.CONTAINER:
		return NewAliCloudProvider(cluster)
	}

	return NewDefaultApplier(cluster)
}
