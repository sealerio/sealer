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

package build

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"

	"github.com/alibaba/sealer/image/cache"

	"github.com/alibaba/sealer/utils/archive"

	"github.com/pkg/errors"

	"github.com/opencontainers/go-digest"

	"github.com/alibaba/sealer/image/store"

	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/command"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/image/reference"
	imageUtils "github.com/alibaba/sealer/image/utils"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/parser"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
)

type Config struct {
	BuildType string
	NoCache   bool
}

type builderLayer struct {
	baseLayers []v1.Layer
	newLayers  []v1.Layer
}

// LocalBuilder: local builder using local provider to build a cluster image
type LocalBuilder struct {
	Config       *Config
	Image        *v1.Image
	Cluster      *v1.Cluster
	ImageNamed   reference.Named
	ImageID      string
	Context      string
	KubeFileName string
	LayerStore   store.LayerStore
	ImageService image.Service
	Prober       image.Prober
	FS           store.Backend
	builderLayer
}

func (l *LocalBuilder) Build(name string, context string, kubefileName string) error {
	err := l.initBuilder(name, context, kubefileName)
	if err != nil {
		return err
	}

	pipLine, err := l.GetBuildPipeLine()
	if err != nil {
		return err
	}

	for _, f := range pipLine {
		if err = f(); err != nil {
			return err
		}
	}
	return nil
}

func (l *LocalBuilder) initBuilder(name string, context string, kubefileName string) error {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return err
	}

	l.ImageNamed = named
	l.Context = context
	l.KubeFileName = kubefileName
	return nil
}

func (l *LocalBuilder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	if err := l.InitImageSpec(); err != nil {
		return nil, err
	}

	buildPipeline = append(buildPipeline,
		l.PullBaseImageNotExist,
		l.ExecBuild,
		l.UpdateImageMetadata)
	return buildPipeline, nil
}

// init default Image metadata
func (l *LocalBuilder) InitImageSpec() error {
	kubeFile, err := utils.ReadAll(l.KubeFileName)
	if err != nil {
		return fmt.Errorf("failed to load kubefile: %v", err)
	}
	l.Image = parser.NewParse().Parse(kubeFile)
	if l.Image == nil {
		return fmt.Errorf("failed to parse kubefile, image is nil")
	}

	layer0 := l.Image.Spec.Layers[0]
	if layer0.Type != common.FROMCOMMAND {
		return fmt.Errorf("first line of kubefile must start with FROM")
	}

	logger.Info("init image spec success!")
	return nil
}

func (l *LocalBuilder) PullBaseImageNotExist() (err error) {
	if l.Image.Spec.Layers[0].Value == common.ImageScratch {
		return nil
	}
	if err = l.ImageService.PullIfNotExist(l.Image.Spec.Layers[0].Value); err != nil {
		return fmt.Errorf("failed to pull baseImage: %v", err)
	}
	logger.Info("pull base image %s success", l.Image.Spec.Layers[0].Value)
	return nil
}

func (l *LocalBuilder) generateSourceFilesDigest(path string) (digest.Digest, error) {
	baseDir := l.Context
	layerDgst, _, err := archive.TarCanonicalDigest(filepath.Join(baseDir, path))
	if err != nil {
		logger.Error(err)
		return "", err
	}
	return layerDgst, nil
}

func (l *LocalBuilder) ExecBuild() error {
	err := l.updateBuilderLayers(l.Image)
	if err != nil {
		return err
	}

	var (
		canUseCache = !l.Config.NoCache
		parentID    = cache.ChainID("")
		newLayers   = l.newLayers
	)

	baseLayerPaths := getBaseLayersPath(l.baseLayers)
	chainSvc, err := cache.NewService()
	if err != nil {
		return err
	}

	for i := 0; i < len(newLayers); i++ {
		layer := &newLayers[i]
		logger.Info("run build layer: %s %s", layer.Type, layer.Value)
		if canUseCache {
			canUseCache, parentID = l.goCache(parentID, layer, chainSvc)
			// cache layer is empty layer
			if canUseCache {
				if layer.ID == "" {
					continue
				}
				baseLayerPaths = append(baseLayerPaths, l.FS.LayerDataDir(layer.ID))
				continue
			}
		}

		if layer.Type == common.COPYCOMMAND {
			err = l.execCopyLayer(layer)
			if err != nil {
				return err
			}
		} else {
			// exec other build cmd,need to mount
			err = l.execOtherLayer(layer, baseLayerPaths)
			if err != nil {
				return err
			}
		}

		if layer.ID == "" {
			continue
		}

		baseLayerPaths = append(baseLayerPaths, l.FS.LayerDataDir(layer.ID))
	}

	logger.Info("exec all build instructs success !")
	return nil
}

// run COPY command, because user can overwrite some file like Cluster file, or build a base image
func (l *LocalBuilder) execCopyLayer(layer *v1.Layer) error {
	//count layer hash;create layer dir ;update image layer hash
	tempDir, err := utils.MkTmpdir()
	if err != nil {
		return fmt.Errorf("failed to create %s:%v", tempDir, err)
	}
	defer utils.CleanDir(tempDir)

	err = l.execLayer(layer, tempDir)
	if err != nil {
		return fmt.Errorf("failed to exec layer %v:%v", layer, err)
	}

	if err = l.calculateLayerDigestAndPlaceIt(layer, tempDir); err != nil {
		return err
	}

	if err = l.SetCacheID(layer); err != nil {
		return err
	}

	return nil
}

//This function only has meaning for copy layers
func (l *LocalBuilder) SetCacheID(layer *v1.Layer) error {
	baseDir := l.Context
	layerDgst, _, err := archive.TarCanonicalDigest(filepath.Join(baseDir, strings.Fields(layer.Value)[0]))
	if err != nil {
		return err
	}

	return l.FS.SetMetadata(layer.ID, cacheID, []byte(layerDgst.String()))
}

func (l *LocalBuilder) squashBaseImageLayerIntoCurrentImage() {
	l.Image.Spec.Layers = append(l.baseLayers, l.newLayers...)
}

func (l *LocalBuilder) execOtherLayer(layer *v1.Layer, lowLayers []string) error {
	tempTarget, err := utils.MkTmpdir()
	if err != nil {
		return fmt.Errorf("failed to create %s:%v", tempTarget, err)
	}
	tempUpper, err := utils.MkTmpdir()
	if err != nil {
		return fmt.Errorf("failed to create %s:%v", tempUpper, err)
	}
	defer utils.CleanDirs(tempTarget, tempUpper)

	if err = l.mountAndExecLayer(layer, tempTarget, tempUpper, lowLayers...); err != nil {
		return err
	}
	if err = l.calculateLayerDigestAndPlaceIt(layer, tempUpper); err != nil {
		return err
	}
	return nil
}

func (l *LocalBuilder) mountAndExecLayer(layer *v1.Layer, tempTarget, tempUpper string, lowLayers ...string) error {
	driver := mount.NewMountDriver()
	err := driver.Mount(tempTarget, tempUpper, lowLayers...)
	if err != nil {
		return fmt.Errorf("failed to mount target %s:%v", tempTarget, err)
	}
	defer func() {
		if err = driver.Unmount(tempTarget); err != nil {
			logger.Warn(fmt.Errorf("failed to umount %s:%v", tempTarget, err))
		}
	}()

	err = l.execLayer(layer, tempTarget)
	if err != nil {
		return fmt.Errorf("failed to exec layer %v:%v", layer, err)
	}
	return nil
}

func (l *LocalBuilder) execLayer(layer *v1.Layer, tempTarget string) error {
	// exec layer cmd;
	if layer.Type == common.COPYCOMMAND {
		src := filepath.Join(l.Context, strings.Fields(layer.Value)[0])
		dest := ""
		if utils.IsDir(src) {
			// src is dir
			dest = filepath.Join(tempTarget, strings.Fields(layer.Value)[1], filepath.Base(src))
		} else {
			// src is file
			dest = filepath.Join(tempTarget, strings.Fields(layer.Value)[1], strings.Fields(layer.Value)[0])
		}
		return utils.RecursionCopy(src, dest)
	}
	if layer.Type == common.RUNCOMMAND || layer.Type == common.CMDCOMMAND {
		cmd := fmt.Sprintf(common.CdAndExecCmd, tempTarget, layer.Value)
		output, err := command.NewSimpleCommand(cmd).Exec()
		logger.Info(output)
		if err != nil {
			return fmt.Errorf("failed to exec %s, err: %v", cmd, err)
		}
	}
	return nil
}

func (l *LocalBuilder) calculateLayerDigestAndPlaceIt(layer *v1.Layer, tempTarget string) error {
	layerDigest, err := l.LayerStore.RegisterLayerForBuilder(tempTarget)
	if err != nil {
		return fmt.Errorf("failed to register layer, err: %v", err)
	}

	layer.ID = layerDigest
	return nil
}

func (l *LocalBuilder) UpdateImageMetadata() error {
	l.setClusterFileToImage()
	l.squashBaseImageLayerIntoCurrentImage()
	err := l.updateImageIDAndSaveImage()
	if err != nil {
		return fmt.Errorf("failed to updateImageIDAndSaveImage, err: %v", err)
	}

	if err = imageUtils.SetImageMetadata(imageUtils.ImageMetadata{
		Name: l.ImageNamed.Raw(),
		ID:   l.Image.Spec.ID,
	}); err != nil {
		return fmt.Errorf("failed to set image metadata :%v", err)
	}
	logger.Info("update image %s to image metadata success !", l.ImageNamed.Raw())
	return nil
}

// setClusterFileToImage: set cluster file whatever build type is
func (l *LocalBuilder) setClusterFileToImage() {
	var clusterFileData string
	if utils.IsFileExist(common.RawClusterfile) {
		bytes, err := utils.ReadAll(common.RawClusterfile)
		if err != nil {
			logger.Warn("failed to set cluster file to image metadata,%s not exist", common.RawClusterfile)
		}
		clusterFileData = string(bytes)
	} else {
		clusterFileData = GetRawClusterFile(l.Image)
	}

	l.addImageAnnotations(common.ImageAnnotationForClusterfile, clusterFileData)
}

func (l *LocalBuilder) updateImageIDAndSaveImage() error {
	imageID, err := generateImageID(*l.Image)
	if err != nil {
		return err
	}

	l.Image.Spec.ID = imageID
	filename := fmt.Sprintf("%s/%s%s", common.DefaultImageDBRootDir, imageID, common.YamlSuffix)

	imageBytes, err := yaml.Marshal(l.Image)
	if err != nil {
		return err
	}

	err = utils.WriteFile(filename, imageBytes)
	if err != nil {
		return err
	}
	logger.Info("write image yaml file to %s success !", filename)
	return nil
}

func generateImageID(image v1.Image) (string, error) {
	imageBytes, err := yaml.Marshal(image)
	if err != nil {
		return "", err
	}
	imageID := digest.FromBytes(imageBytes).Hex()
	return imageID, nil
}

// GetClusterFile from user build context or from base image
func GetRawClusterFile(im *v1.Image) string {
	if im.Spec.Layers[0].Value == common.ImageScratch {
		data, err := ioutil.ReadFile(filepath.Join("etc", common.DefaultClusterFileName))
		if err != nil {
			return ""
		}
		return string(data)
	}
	// find cluster file from context
	if clusterFile := getClusterFileFromContext(im); clusterFile != nil {
		logger.Info("get cluster file from context success!")
		return string(clusterFile)
	}
	// find cluster file from base image
	clusterFile := image.GetClusterFileFromImage(im.Spec.Layers[0].Value)
	if clusterFile != "" {
		logger.Info("get cluster file from base image success!")
		return clusterFile
	}
	return ""
}

func getClusterFileFromContext(image *v1.Image) []byte {
	for i := range image.Spec.Layers {
		layer := image.Spec.Layers[i]
		if layer.Type == common.COPYCOMMAND && strings.Fields(layer.Value)[0] == common.DefaultClusterFileName {
			if clusterFile, _ := utils.ReadAll(strings.Fields(layer.Value)[0]); clusterFile != nil {
				return clusterFile
			}
		}
	}
	return nil
}

// GetClusterFile from user build context or from base image
func (l *LocalBuilder) addImageAnnotations(key, value string) {
	if l.Image.Annotations == nil {
		l.Image.Annotations = make(map[string]string)
	}
	l.Image.Annotations[key] = value
}

func (l *LocalBuilder) updateBuilderLayers(image *v1.Image) error {
	// we do not check the len of layers here, because we checked it before.
	// remove the first layer of image
	var (
		layer0    = image.Spec.Layers[0]
		baseImage *v1.Image
		err       error
	)

	// and the layer 0 must be from layer
	if layer0.Value == common.ImageScratch {
		// give a empty image
		baseImage = &v1.Image{}
	} else {
		baseImage, err = imageUtils.GetImage(image.Spec.Layers[0].Value)
		if err != nil {
			return fmt.Errorf("failed to get base image while updating base layers, err: %s", err)
		}
	}

	l.baseLayers = append([]v1.Layer{}, baseImage.Spec.Layers...)
	l.newLayers = append([]v1.Layer{}, image.Spec.Layers[1:]...)
	if len(l.baseLayers)+len(l.newLayers) > maxLayerDeep {
		return errors.New("current number of layers exceeds 128 layers")
	}
	return nil
}

func (l *LocalBuilder) goCache(parentID cache.ChainID, layer *v1.Layer, cacheService cache.Service) (continueCache bool, chainID cache.ChainID) {
	var (
		srcDigest = digest.Digest("")
		err       error
	)

	// specially for copy command, we would generate digest of src file as srcDigest.
	// and use srcDigest as cacheID to generate a cacheLayer, eventually use the cacheLayer
	// to hit the cache layer
	if layer.Type == common.COPYCOMMAND {
		srcDigest, err = l.generateSourceFilesDigest(strings.Fields(layer.Value)[0])
		if err != nil {
			logger.Warn("failed to generate src digest, discard cache, err: %s", err)
		}
	}

	cacheLayer := cacheService.NewCacheLayer(*layer, srcDigest)
	cacheLayerID, err := l.Prober.Probe(parentID.String(), &cacheLayer)
	if err != nil {
		logger.Debug("failed to probe cache for %+v, err: %s", layer, err)
		return false, ""
	}
	// cache hit
	logger.Info("---> Using cache %v", cacheLayerID)
	layer.ID = cacheLayerID
	cID, err := cacheLayer.ChainID(parentID)
	if err != nil {
		return false, ""
	}
	return true, cID
}

// used in build stage, where the image still has from layer
func getBaseLayersPath(layers []v1.Layer) (res []string) {
	for _, layer := range layers {
		if layer.ID != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.ID.Hex()))
		}
	}
	return res
}

func NewLocalBuilder(config *Config) (Interface, error) {
	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, err
	}

	service := image.NewImageService()
	fs, err := store.NewFSStoreBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to init store backend, err: %s", err)
	}

	prober := image.NewImageProber(service, config.NoCache)
	return &LocalBuilder{
		Config:       config,
		LayerStore:   layerStore,
		ImageService: service,
		Prober:       prober,
		FS:           fs,
		builderLayer: builderLayer{
			// for skip golang ci
			baseLayers: []v1.Layer{},
			newLayers:  []v1.Layer{},
		},
	}, nil
}
