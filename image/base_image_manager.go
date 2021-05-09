package image

import (
	"context"
	"encoding/json" //nolint:goimports
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"
	pkgutils "github.com/alibaba/sealer/utils"
	"github.com/docker/docker/api/types"
	"github.com/justadogistaken/reg/registry"
	"github.com/opencontainers/go-digest"
	"github.com/wonderivan/logger"
	"sigs.k8s.io/yaml"
)

// BaseImageManager take the responsibility to store common values
type BaseImageManager struct {
	registry *registry.Registry
}

func (bim BaseImageManager) syncImageLocal(image v1.Image) (err error) {
	err = syncImage(image)
	if err != nil {
		return err
	}

	err = syncImagesMap(image)
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

func (bim BaseImageManager) deleteImageLocal(imageName, imageID string) (err error) {
	// Read image metadata from file to ensure that if we fail to delete image records,
	// the image metadata can be recovered from it.
	image, err := imageutils.GetImage(imageName)
	if err != nil {
		return err
	}

	err = deleteImage(imageID)
	if err != nil {
		return err
	}

	err = imageutils.DeleteImage(imageName)
	if err != nil {
		err = syncImage(*image)
		if err != nil {
			return fmt.Errorf("failed to delete image records in %s and failed to recover image metadata file: %s",
				common.DefaultImageMetadataFile, filepath.Join(common.DefaultImageMetaRootDir, imageID+common.YamlSuffix))
		}
		return err
	}

	return nil
}

// init bim registry
func (bim *BaseImageManager) initRegistry(hostname string) error {
	var (
		dockerInfo       *pkgutils.DockerInfo
		err              error
		username, passwd string
	)
	dockerInfo, err = pkgutils.DockerConfig()
	if err != nil {
		logger.Warn("failed to get docker info, err: %s", err)
	} else {
		username, passwd, err = dockerInfo.DecodeDockerAuth(hostname)
		if err != nil {
			logger.Warn("failed to decode auth info, username and password would be empty, err: %s", err)
		}
	}

	var reg *registry.Registry
	reg, err = bim.fetchRegistryClient(types.AuthConfig{ServerAddress: hostname, Username: username, Password: passwd})
	if nil != err {
		return err
	}

	bim.registry = reg
	return nil
}

//fetch https and http registry client
func (bim *BaseImageManager) fetchRegistryClient(auth types.AuthConfig) (*registry.Registry, error) {
	reg, err := registry.New(context.Background(), auth, registry.Opt{Insecure: true})
	if err == nil {
		return reg, nil
	}
	reg, err = registry.New(context.Background(), auth, registry.Opt{Insecure: true, NonSSL: true})
	if err == nil {
		return reg, nil
	}
	return nil, err
}

func (bim BaseImageManager) downloadImageManifestConfig(named reference.Named, dig digest.Digest) (v1.Image, error) {
	// download image metadata
	configReader, err := bim.registry.DownloadLayer(context.Background(), named.Repo(), dig)
	if err != nil {
		return v1.Image{}, err
	}
	decoder := json.NewDecoder(configReader)

	var img v1.Image
	return img, decoder.Decode(&img)
}

// used to sync image into DefaultImageMetadataFile
func syncImagesMap(image v1.Image) error {
	return imageutils.SetImageMetadata(imageutils.ImageMetadata{Name: image.Name, ID: image.Spec.ID})
}

// dump image yaml to DefaultImageMetaRootDir
func syncImage(image v1.Image) error {
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
