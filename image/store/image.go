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

package store

import (
	"io/ioutil"
	"os"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"
	pkgutils "github.com/alibaba/sealer/utils"

	"path/filepath"

	"sigs.k8s.io/yaml"
)

func SyncImageLocal(image v1.Image, named reference.Named) (err error) {
	err = saveImage(image)
	if err != nil {
		return err
	}

	err = syncImagesMap(named.Raw(), image.Spec.ID)
	if err != nil {
		// this won't fail literally
		if err = os.Remove(filepath.Join(common.DefaultImageMetaRootDir,
			image.Spec.ID+common.YamlSuffix)); err != nil {
			return err
		}
		return err
	}
	return nil
}

func DeleteImageLocal(imageID string) (err error) {
	return deleteImage(imageID)
}

// used to sync image into DefaultImageMetadataFile
func syncImagesMap(name, id string) error {
	return imageutils.SetImageMetadata(imageutils.ImageMetadata{Name: name, ID: id})
}

// dump image yaml to DefaultImageMetaRootDir
func saveImage(image v1.Image) error {
	imageYaml, err := yaml.Marshal(image)
	if err != nil {
		return err
	}

	err = pkgutils.MkDirIfNotExists(common.DefaultImageMetaRootDir)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(common.DefaultImageMetaRootDir, image.Spec.ID+common.YamlSuffix), imageYaml, common.FileMode0644)
}

func deleteImage(imageID string) error {
	file := filepath.Join(common.DefaultImageMetaRootDir, imageID+common.YamlSuffix)
	if pkgutils.IsFileExist(file) {
		err := pkgutils.CleanFiles(file)
		if err != nil {
			return err
		}
	}
	return nil
}
