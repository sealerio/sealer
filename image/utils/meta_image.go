package utils

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"io/ioutil"
	"path/filepath"
)

const DefaultJsonIndent = "  "

type ImageMetadataMap map[string]ImageMetadata

type ImageMetadata struct {
	Name string `json:"name,omitempty"`
	Id   string `json:"id,omitempty"`
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

	if image.Id == "" {
		return nil, fmt.Errorf("failed to find corresponding image id, id is empty")
	}

	return GetImageById(image.Id)
}

func GetImageById(imageId string) (*v1.Image, error) {
	fileName := filepath.Join(common.DefaultImageMetaRootDir, imageId+".yaml")

	var image v1.Image
	if err := utils.UnmarshalYamlFile(fileName, &image); err != nil {
		return nil, fmt.Errorf("%s.yaml parsing failed, %s", imageId, err)
	}

	return &image, nil
}

//get all imageMetadata
func GetImageMetadataMap() (ImageMetadataMap, error) {
	// create file if not exists
	if !utils.IsFileExist(common.DefaultImageMetadataFile) {
		if err := utils.WriteFile(common.DefaultImageMetadataFile, []byte("{}")); err != nil {
			return nil, err
		} else {
			return ImageMetadataMap{}, nil
		}
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

//set one imageMetadata to imageMetadataMap
func SetImageMetadata(metadata ImageMetadata) error {
	imagesMap, err := GetImageMetadataMap()
	if err != nil {
		return err
	}

	imagesMap[metadata.Name] = metadata
	data, err := json.MarshalIndent(imagesMap, "", DefaultJsonIndent)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(common.DefaultImageMetadataFile, data, common.FileMode0644); err != nil {
		return errors.Wrap(err, "failed to write DefaultImageMetadataFile")
	}
	return nil
}
