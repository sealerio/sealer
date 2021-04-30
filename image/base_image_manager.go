package image

import (
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/justadogistaken/reg/registry"
	"github.com/opencontainers/go-digest"
	"github.com/wonderivan/logger"
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"gitlab.alibaba-inc.com/seadent/pkg/image/reference"
	imageutils "gitlab.alibaba-inc.com/seadent/pkg/image/utils"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	pkgutils "gitlab.alibaba-inc.com/seadent/pkg/utils"
	"io/ioutil"
	"os"
	"path/filepath"
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

	reg, err := registry.New(context.Background(), types.AuthConfig{ServerAddress: hostname, Username: username, Password: passwd}, registry.Opt{Insecure: true})
	if err != nil {
		return err
	}

	bim.registry = reg
	return nil
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
	return imageutils.SetImageMetadata(imageutils.ImageMetadata{Name: image.Name, Id: image.Spec.ID})
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

	return ioutil.WriteFile(filepath.Join(common.DefaultImageMetaRootDir, image.Spec.ID+common.YamlSuffix), imageYaml, common.FileMode0766)
}
