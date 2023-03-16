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
	"fmt"

	"github.com/containers/buildah"
	"github.com/sealerio/sealer/cmd/sealer/cmd/alpha"
	"github.com/sealerio/sealer/pkg/define/options"
	imagebuildah "github.com/sealerio/sealer/pkg/imageengine/buildah"
	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/utils/os"
)

func Mount(mountInfo alpha.MountService, name string) error {
	mountPoint, err := mountInfo.Mount(name)
	testhelper.CheckErr(err)

	if ok := os.IsDir(mountPoint); !ok {
		return fmt.Errorf("this directory does not exist")
	}
	return nil
}

func GetContainerID() (string, error) {
	engine, err := imagebuildah.NewBuildahImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		testhelper.CheckErr(err)
	}
	store := engine.ImageStore()
	clients, err := buildah.OpenAllBuilders(store)
	if err != nil {
		testhelper.CheckErr(err)
	}
	for _, client := range clients {
		mounted, err := client.Mounted()
		if err != nil {
			testhelper.CheckErr(err)
		}
		if mounted {
			return client.ContainerID, nil
		}
	}
	return "", nil
}
