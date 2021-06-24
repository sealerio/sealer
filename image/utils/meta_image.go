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

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/pkg/errors"
)

const DefaultJSONIndent = "\t"

type ImageMetadataMap map[string]ImageMetadata

type ImageMetadata struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

func GetImage(imageName string) (*v1.Image, error) {
	imagesMap, err := GetImageMetadataMap()
	if err != nil {
		return nil, err
	}
	//get an imageId based on the name of ClusterImage
	image, ok := imagesMap[imageName]
	if !ok {
		return nil, fmt.Errorf("failed to find image by name: %s", imageName)
	}

	if image.ID == "" {
		return nil, fmt.Errorf("failed to find corresponding image id, id is empty")
	}

	return GetImageByID(image.ID)
}

func DeleteImage(imageName string) error {
	imagesMap, err := GetImageMetadataMap()
	if err != nil {
		return err
	}

	_, ok := imagesMap[imageName]
	if !ok {
		return nil
	}
	delete(imagesMap, imageName)

	data, err := json.MarshalIndent(imagesMap, "", DefaultJSONIndent)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(common.DefaultImageMetadataFile, data, common.FileMode0644); err != nil {
		return errors.Wrap(err, "failed to write DefaultImageMetadataFile")
	}
	return nil
}

func DeleteImageByID(imageID string, force bool) error {
	imagesMap, err := GetImageMetadataMap()
	if err != nil {
		return err
	}
	var imageIDCount = 0
	var imageNames []string
	for _, value := range imagesMap {
		if value.ID == imageID {
			imageIDCount++
			imageNames = append(imageNames, value.Name)
		}
		if imageIDCount > 1 && !force {
			return fmt.Errorf("there are more than one image %s", imageID)
		}
	}
	if imageIDCount == 0 {
		return fmt.Errorf("failed to find image with id %s", imageID)
	}
	for _, imageName := range imageNames {
		delete(imagesMap, imageName)
	}

	data, err := json.MarshalIndent(imagesMap, "", DefaultJSONIndent)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(common.DefaultImageMetadataFile, data, common.FileMode0644); err != nil {
		return errors.Wrap(err, "failed to write DefaultImageMetadataFile")
	}
	return nil
}

func GetImageByID(imageID string) (*v1.Image, error) {
	fileName := filepath.Join(common.DefaultImageDBRootDir, imageID+".yaml")

	var image v1.Image
	if err := utils.UnmarshalYamlFile(fileName, &image); err != nil {
		return nil, fmt.Errorf("%s.yaml parsing failed, %s", imageID, err)
	}

	return &image, nil
}

func GetImageMetadataMap() (ImageMetadataMap, error) {
	// create file if not exists
	if !utils.IsFileExist(common.DefaultImageMetadataFile) {
		if err := utils.WriteFile(common.DefaultImageMetadataFile, []byte("{}")); err != nil {
			return nil, err
		}
		return ImageMetadataMap{}, nil
	}

	data, err := ioutil.ReadFile(common.DefaultImageMetadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read ImageMetadataMap, err: %s", err)
	}

	var imagesMap ImageMetadataMap
	if err = json.Unmarshal(data, &imagesMap); err != nil {
		return nil, fmt.Errorf("failed to parsing ImageMetadataMap, err: %s", err)
	}
	return imagesMap, err
}

func SetImageMetadata(metadata ImageMetadata) error {
	imagesMap, err := GetImageMetadataMap()
	if err != nil {
		return err
	}

	imagesMap[metadata.Name] = metadata
	data, err := json.MarshalIndent(imagesMap, "", DefaultJSONIndent)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(common.DefaultImageMetadataFile, data, common.FileMode0644); err != nil {
		return errors.Wrap(err, "failed to write DefaultImageMetadataFile")
	}
	return nil
}

func GetImageMetadata(imageNameOrID string) (ImageMetadata, error) {
	imageMetadataMap, err := GetImageMetadataMap()
	imageMetadata := ImageMetadata{}
	if err != nil {
		return imageMetadata, err
	}
	for k, v := range imageMetadataMap {
		if imageNameOrID == k || imageNameOrID == v.ID {
			return v, nil
		}
	}
	return imageMetadata, &ImageNameOrIDNotFoundError{name: imageNameOrID}
}
