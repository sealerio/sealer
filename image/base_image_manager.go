package image

import (
	"context"
	"encoding/json" //nolint:goimports
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"
	pkgutils "github.com/alibaba/sealer/utils"
	"github.com/docker/docker/api/types"
	"github.com/justadogistaken/reg/registry"
	"github.com/opencontainers/go-digest"

	"path/filepath"

	"sigs.k8s.io/yaml"
)

// BaseImageManager take the responsibility to store common values
type BaseImageManager struct {
	registry *registry.Registry
}

func (bim BaseImageManager) syncImageLocal(image v1.Image, named reference.Named) (err error) {
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

func (bim BaseImageManager) deleteImageLocal(imageID string) (err error) {
	return deleteImage(imageID)
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
		return fmt.Errorf("failed to get docker info, err: %s", err)
	} else {
		username, passwd, err = dockerInfo.DecodeDockerAuth(hostname)
		if err != nil {
			return fmt.Errorf("failed to decode auth info, username and password would be empty, err: %s", err)
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
