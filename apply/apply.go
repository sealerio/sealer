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
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/pkg/image/store"

	"github.com/alibaba/sealer/apply/applydriver"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/filesystem"
	"github.com/alibaba/sealer/pkg/image"
	v2 "github.com/alibaba/sealer/types/api/v2"
)

func NewApplierFromFile(clusterfile string) (applydriver.Interface, error) {
	clusterData, err := ioutil.ReadFile(filepath.Clean(clusterfile))
	if err != nil {
		return nil, err
	}
	cluster, err := GetClusterFromDataCompatV1(string(clusterData))
	if err != nil {
		return nil, err
	}
	if cluster.Name == "" {
		return nil, fmt.Errorf("cluster name cannot be empty, make sure %s file is correct", clusterfile)
	}
	cluster.SetAnnotations(common.ClusterfileName, clusterfile)
	return NewApplier(cluster)
}

func NewApplier(cluster *v2.Cluster) (applydriver.Interface, error) {
	/*	switch cluster.Spec.Provider {
		case common.AliCloud:
			return NewAliCloudProvider(cluster)
		case common.CONTAINER:
			return NewAliCloudProvider(cluster)
		}*/
	return NewDefaultApplier(cluster)
}

/*func NewAliCloudProvider(cluster *v2.Cluster) (applydriver.Interface, error) {
	return &applydriver.CloudApplier{
		ClusterDesired: cluster,
	}, nil
}*/

func NewDefaultApplier(cluster *v2.Cluster) (applydriver.Interface, error) {
	imgSvc, err := image.NewImageService()
	if err != nil {
		return nil, err
	}

	mounter, err := filesystem.NewCloudImageMounter()
	if err != nil {
		return nil, err
	}

	is, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return &applydriver.Applier{
		ClusterDesired:    cluster,
		ImageManager:      imgSvc,
		CloudImageMounter: mounter,
		ImageStore:        is,
	}, nil
}
