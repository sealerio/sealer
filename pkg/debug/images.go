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

package debug

import (
	"fmt"
)

type ImagesManagement interface {
	ShowDefaultImages() error
	GetDefaultImage() (string, error)
}

const DefaultSealerRegistryURL = "registry.cn-qingdao.aliyuncs.com/sealer-apps/"

// ImagesManager holds the default images information.
type ImagesManager struct {
	RegistryURL string

	DefaultImagesMap map[string]string // "RichToolsOnUbuntu": "debug:ubuntu"
	DefaultImageKey  string            // "RichToolsOnUbuntu"

	DefaultImage string // RegistryURL + DefaultImagesMap[DefaultImageName]
}

func NewDebugImagesManager() *ImagesManager {
	return &ImagesManager{
		DefaultImagesMap: map[string]string{
			"RichToolsOnUbuntu": "debug:ubuntu",
		},

		DefaultImageKey: "RichToolsOnUbuntu",
	}
}

// ShowDefaultImages shows default images provided by debug.
func (manager *ImagesManager) ShowDefaultImages() error {
	if len(manager.RegistryURL) == 0 {
		manager.RegistryURL = DefaultSealerRegistryURL
	}
	fmt.Println("There are several default images you can use：")
	for key, value := range manager.DefaultImagesMap {
		fmt.Println(key + ":  " + manager.RegistryURL + value)
	}

	return nil
}

// GetDefaultImage return the default image provide by debug.
func (manager *ImagesManager) GetDefaultImage() (string, error) {
	if len(manager.RegistryURL) == 0 {
		manager.RegistryURL = DefaultSealerRegistryURL
	}
	return manager.RegistryURL + manager.DefaultImagesMap[manager.DefaultImageKey], nil
}
