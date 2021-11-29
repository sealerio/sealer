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

	"github.com/alibaba/sealer/apply/applytype"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/image"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

func NewApplierFromFile(clusterfile string) (applytype.Interface, error) {
	clusters, err := utils.DecodeCluster(clusterfile)
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return nil, fmt.Errorf("failed to found cluster from %s", clusterfile)
	}
	if len(clusters) > 1 {
		return nil, fmt.Errorf("multiple clusters exist in the Clusterfile")
	}
	cluster := &clusters[0]
	cluster.SetAnnotations(common.ClusterfileName, clusterfile)
	return NewApplier(cluster)
}

func NewApplier(cluster *v1.Cluster) (applytype.Interface, error) {
	switch cluster.Spec.Provider {
	case common.AliCloud:
		return NewAliCloudProvider(cluster)
	case common.CONTAINER:
		return NewAliCloudProvider(cluster)
	}
	return NewDefaultApplier(cluster)
}

func NewAliCloudProvider(cluster *v1.Cluster) (applytype.Interface, error) {
	return &applytype.CloudApplier{
		ClusterDesired: cluster,
	}, nil
}

func NewDefaultApplier(cluster *v1.Cluster) (applytype.Interface, error) {
	imgSvc, err := image.NewImageService()
	if err != nil {
		return nil, err
	}

	fs, err := filesystem.NewFilesystem()
	if err != nil {
		return nil, err
	}

	return &applytype.Applier{
		ClusterDesired: cluster,
		ImageManager:   imgSvc,
		FileSystem:     fs,
	}, nil
}
