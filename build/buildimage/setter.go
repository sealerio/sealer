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

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type annotation struct {
}

func (a annotation) Set(ima *v1.Image) error {
	return a.setClusterFile(ima)
}

func (a annotation) setClusterFile(ima *v1.Image) error {
	var (
		err error
	)

	cluster := &v2.Cluster{}
	cluster.Kind = common.Kind
	cluster.APIVersion = common.APIVersion
	cluster.Name = "my-cluster"
	cluster.Spec.SSH = v1.SSH{
		Port: "22",
		User: "root",
	}

	cluster.Spec.Image = ima.Name
	err = setClusterFileToImage(cluster, ima)
	if err != nil {
		return fmt.Errorf("failed to set image metadata, err: %v", err)
	}
	return nil
}

func NewAnnotationSetter() ImageSetter {
	return annotation{}
}
