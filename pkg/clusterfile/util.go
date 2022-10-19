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

package clusterfile

import (
	"fmt"

	"github.com/sealerio/sealer/common"
	v2 "github.com/sealerio/sealer/types/api/v2"
	yamlUtils "github.com/sealerio/sealer/utils/yaml"
)

func GetClusterFromFile(filepath string) (cluster *v2.Cluster, err error) {
	cluster = &v2.Cluster{}
	if err = yamlUtils.UnmarshalFile(filepath, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster from %s: %v", filepath, err)
	}
	cluster.SetAnnotations(common.ClusterfileName, filepath)
	return cluster, nil
}
