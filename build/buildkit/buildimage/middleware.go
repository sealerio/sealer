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
	"github.com/alibaba/sealer/pkg/image/save"
	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/utils"
)

var (
	imageListWithAuth = "imageListWithAuth.yaml"
)

type MiddlewarePuller struct {
	puller save.DefaultImageSaver
}

func (s MiddlewarePuller) Process(context, rootfs string) error {
	//read the filePath named "imageListWithAuth.yaml" if not exists just return;
	//pares the images and save to rootfs
	filePath := filepath.Join(context, imageListWithAuth)
	if !utils.IsExist(filePath) {
		return nil
	}

	// pares middleware file: imageListWithAuth.yaml
	var imageSection []save.ImageSection
	ia := make(save.ImageListWithAuth)

	err := utils.UnmarshalYamlFile(filePath, &imageSection)
	if err != nil {
		return err
	}

	for _, section := range imageSection {
		if len(section.Images) == 0 {
			continue
		}
		if section.Username == "" || section.Password == "" {
			return fmt.Errorf("must set username and password at imageListWithAuth.yaml")
		}
		auth, nameds, err := save.NewImageListWithAuth(section)
		if err != nil {
			return err
		}
		domainToImages := make(map[string][]save.Named)
		for _, named := range nameds {
			domainToImages[named.Domain()+named.Repo()] = append(domainToImages[named.Domain()+named.Repo()], named)
		}

		ia[auth] = domainToImages
	}

	if len(ia) == 0 {
		return nil
	}

	plat := runtime.GetCloudImagePlatform(rootfs)
	return s.puller.SaveImagesWithAuth(ia, filepath.Join(rootfs, common.RegistryDirName), plat)
}

func NewMiddlewarePuller() Middleware {
	return MiddlewarePuller{
		puller: save.DefaultImageSaver{},
	}
}
