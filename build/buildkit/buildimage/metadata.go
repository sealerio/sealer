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

package buildimage

import (
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/build/buildkit/buildinstruction"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/utils"
)

type metadata struct {
}

func (m metadata) Process(src, dst buildinstruction.MountTarget) error {
	srcPath := src.GetMountTarget()
	rootfs := dst.GetMountTarget()
	// if Metadata file existed in srcPath, load and marshal to check the legality of it's content.
	// if not, use rootfs Metadata.
	smd, err := runtime.LoadMetadata(srcPath)
	if err != nil {
		return err
	}
	if smd != nil {
		return nil
	}

	md, err := runtime.LoadMetadata(rootfs)
	if err != nil {
		return err
	}
	if md == nil {
		return fmt.Errorf("failed to load rootfs Metadata, err: %v", err)
	}

	kv := getKubeVersion(srcPath)
	if md.KubeVersion == kv {
		return nil
	}

	md.KubeVersion = kv
	mf := filepath.Join(rootfs, common.DefaultMetadataName)
	if err = utils.MarshalJSONToFile(mf, md); err != nil {
		return fmt.Errorf("failed to set image Metadata file, err: %v", err)
	}

	return nil
}

func NewMetadataDiffer() Differ {
	return metadata{}
}
