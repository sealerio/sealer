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

package buildimage

import (
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
)

type annotation struct {
	source string
}

func (a annotation) Set(ima *v1.Image) error {
	return a.setClusterFile(ima)
}

func (a annotation) setClusterFile(ima *v1.Image) error {
	var (
		err      error
		filePath = filepath.Join(a.source, "etc", common.DefaultClusterFileName)
	)

	cluster := &v2.Cluster{}
	cluster.Kind = common.Kind
	cluster.APIVersion = common.APIVersion
	cluster.Name = "my-cluster"
	cluster.Spec.SSH = v1.SSH{
		Port: "22",
		User: "root",
	}
	// if rootfs has Clusterfile, load it.
	if utils.IsExist(filePath) {
		cluster, err = LoadClusterFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to load clusterfile, err: %v", err)
		}
	}

	cluster.Spec.Image = ima.Name
	err = setClusterFileToImage(cluster, ima)
	if err != nil {
		return fmt.Errorf("failed to set image metadata, err: %v", err)
	}
	return nil
}

func NewAnnotationSetter(rootfs string) ImageSetter {
	return annotation{
		source: rootfs,
	}
}

type platform struct {
	plat v1.Platform
}

func (p platform) Set(ima *v1.Image) error {
	ima.Spec.Platform = p.plat
	return nil
}

func NewPlatformSetter(plat v1.Platform) ImageSetter {
	return platform{
		plat: plat,
	}
}
