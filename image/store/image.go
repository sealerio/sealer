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

	return ioutil.WriteFile(filepath.Join(common.DefaultImageMetaRootDir, image.Spec.ID+common.YamlSuffix), imageYaml, common.FileMode0755)
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
