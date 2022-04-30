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

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/image/save"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils"
)

var (
	imageListWithAuth = "imageListWithAuth.yaml"
)

type ImageSection struct {
	Registry string   `json:"registry,omitempty"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Images   []string `json:"images,omitempty"`
}

type MiddlewarePuller struct {
	puller   save.DefaultImageSaver
	platform v1.Platform
}

func (m MiddlewarePuller) Process(context, rootfs string) error {
	//read the filePath named "imageListWithAuth.yaml" if not exists just return;
	//pares the images and save to rootfs
	filePath := filepath.Join(context, imageListWithAuth)
	if !utils.IsExist(filePath) {
		return nil
	}

	// pares middleware file: imageListWithAuth.yaml
	var imageSection []ImageSection
	err := utils.UnmarshalYamlFile(filePath, &imageSection)
	if err != nil {
		return err
	}

	ia := make(save.ImageListWithAuth, 0)
	for _, section := range imageSection {
		if len(section.Images) == 0 {
			continue
		}
		if section.Username == "" || section.Password == "" {
			return fmt.Errorf("must set username and password at imageListWithAuth.yaml")
		}

		domainToImages, err := normalizedImageListWithAuth(section)
		if err != nil {
			return err
		}

		ia = append(ia, save.Section{
			Registry: section.Registry,
			Username: section.Username,
			Password: section.Password,
			Images:   domainToImages,
		})
	}

	if len(ia) == 0 {
		return nil
	}

	return m.puller.SaveImagesWithAuth(ia, filepath.Join(rootfs, common.RegistryDirName), m.platform)
}

func normalizedImageListWithAuth(sec ImageSection) (map[string][]save.Named, error) {
	domainToImages := make(map[string][]save.Named)
	for _, image := range sec.Images {
		named, err := save.ParseNormalizedNamed(image, sec.Registry)
		if err != nil {
			return nil, fmt.Errorf("parse image name error: %v", err)
		}
		domainToImages[named.Domain()+named.Repo()] = append(domainToImages[named.Domain()+named.Repo()], named)
	}
	return domainToImages, nil
}

func NewMiddlewarePuller(platform v1.Platform) Differ {
	return MiddlewarePuller{
		platform: platform,
		puller:   save.DefaultImageSaver{},
	}
}
