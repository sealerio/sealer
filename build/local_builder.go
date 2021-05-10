package build

import (
	"fmt"
	"io/ioutil"
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
	"github.com/alibaba/sealer/utils/hash"
	"github.com/alibaba/sealer/utils/mount"
)

type Config struct {
}

// LocalBuilder: local builder using local provider to build a cluster image
type LocalBuilder struct {
	Config       *Config
	Image        *v1.Image
	Cluster      *v1.Cluster
	ImageName    string
	ImageID      string
	Context      string
	KubeFileName string
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

	l.ImageName = named.Raw()
	l.Context = context
	l.KubeFileName = kubefileName
	return nil
}

func (l *LocalBuilder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	if err := l.InitImageSpec(); err != nil {
		return nil, err
	}
	if l.IsOnlyCopy() {
		buildPipeline = append(buildPipeline,
			l.ExecBuild,
			l.UpdateImageMetadata,
			l.PushToRegistry)
	} else {
		buildPipeline = append(buildPipeline,
			l.PullBaseImageNotExist,
			l.ApplyCluster,
			l.ExecBuild,
			l.UpdateImageMetadata,
			l.PushToRegistry)
	}
	return buildPipeline, nil
}

// init default Image metadata
func (l *LocalBuilder) InitImageSpec() error {
	kubeFile, err := utils.ReadAll(l.KubeFileName)
	if err != nil {
		return fmt.Errorf("failed to load kubefile: %v", err)
	}
	l.Image = parser.NewParse().Parse(kubeFile, l.ImageName)
	if l.Image == nil {
		return fmt.Errorf("failed to parse kubefile, image is nil")
	}

	layer0 := l.Image.Spec.Layers[0]
	if layer0.Type != common.FROMCOMMAND {
		return fmt.Errorf("first line of kubefile must be FROM")
	}

	logger.Info("init image spec success! image id is %s", l.Image.Spec.ID)
	return nil
}
func (l *LocalBuilder) IsOnlyCopy() bool {
	for i := 1; i < len(l.Image.Spec.Layers); i++ {
		if l.Image.Spec.Layers[i].Type == common.RUNCOMMAND ||
			l.Image.Spec.Layers[i].Type == common.CMDCOMMAND {
			return false
		}
	}
	return true
}

func (l *LocalBuilder) PullBaseImageNotExist() (err error) {
	if l.Image.Spec.Layers[0].Value == common.ImageScratch {
		return nil
	}
	if err = image.NewImageService().PullIfNotExist(l.Image.Spec.Layers[0].Value); err != nil {
		return fmt.Errorf("failed to pull baseImage: %v", err)
	}
	logger.Info("pull baseImage %s success", l.Image.Spec.Layers[0].Value)
	return nil
}

func (l *LocalBuilder) ExecBuild() error {
	baseLayers, err := getBaseLayersFromImage(*l.Image)
	if err != nil {
		return err
	}
	// TODO a little bit confused about the block, jiangnan
	for i := 1; i < len(l.Image.Spec.Layers); i++ {
		layer := &l.Image.Spec.Layers[i]
		logger.Info("run build layer: %s %s", layer.Type, layer.Value)
		if layer.Type == common.COPYCOMMAND {
			err = l.execCopyLayer(layer)
			if err != nil {
				return err
			}
		} else {
			// exec other build cmd,need to mount
			err = l.execOtherLayer(layer, baseLayers)
			if err != nil {
				return err
			}
		}
		baseLayers = append(baseLayers, filepath.Join(common.DefaultLayerDir, layer.Hash))
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
	if err = l.countLayerHash(layer, tempDir); err != nil {
		return err
	}
	return nil
}

func (l *LocalBuilder) squashBaseImageLayerIntoCurrentImage() (err error) {
	if len(l.Image.Spec.Layers) == 0 || l.Image.Spec.Layers[0].Type != common.FROMCOMMAND {
		return nil
	}

	value := l.Image.Spec.Layers[0].Value
	l.Image.Spec.Layers = l.Image.Spec.Layers[1:]
	if value == common.ImageScratch {
		return nil
	}

	img, err := image.NewImageMetadataService().GetImage(value)
	if err != nil {
		return err
	}

	l.Image.Spec.Layers = append(img.Spec.Layers, l.Image.Spec.Layers...)
	return nil
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
	if err = l.countLayerHash(layer, tempUpper); err != nil {
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
	err = l.execLayer(layer, tempTarget)
	if err != nil {
		return fmt.Errorf("failed to exec layer %v:%v", layer, err)
	}
	if err = driver.Unmount(tempTarget); err != nil {
		return fmt.Errorf("failed to umount %s:%v", tempTarget, err)
	}
	return nil
}

func (l *LocalBuilder) execLayer(layer *v1.Layer, tempTarget string) error {
	// exec layer cmd;
	if layer.Type == common.COPYCOMMAND {
		dist := ""
		if utils.IsDir(strings.Fields(layer.Value)[0]) {
			// src is dir
			dist = filepath.Join(tempTarget, strings.Fields(layer.Value)[1])
		} else {
			// src is file
			dist = filepath.Join(tempTarget, strings.Fields(layer.Value)[1], strings.Fields(layer.Value)[0])
		}
		return utils.RecursionCopy(strings.Fields(layer.Value)[0], dist)
	}
	if layer.Type == common.RUNCOMMAND || layer.Type == common.CMDCOMMAND {
		cmd := fmt.Sprintf(common.CdAndExecCmd, tempTarget, layer.Value)
		_, err := command.NewSimpleCommand(cmd).Exec()
		return err
	}
	return nil
}

func (l *LocalBuilder) countLayerHash(layer *v1.Layer, tempTarget string) error {
	layerHash, err := hash.CheckSumAndPlaceLayer(tempTarget)
	if err != nil {
		return fmt.Errorf("failed to count layer hash:%v", err)
	}
	emptyHash := hash.SHA256{}.EmptyDigest().Hex()
	if layerHash == emptyHash {
		layerHash = ""
	}
	layer.Hash = layerHash
	return nil
}

func (l *LocalBuilder) ApplyCluster() error {
	return l.applyCluster()
}

func (l *LocalBuilder) UpdateImageMetadata() error {
	if err := l.squashBaseImageLayerIntoCurrentImage(); err != nil {
		return err
	}
	// write image info to its metadata
	filename := fmt.Sprintf("%s/%s%s", common.DefaultImageMetaRootDir, l.Image.Spec.ID, common.YamlSuffix)
	//set cluster file
	if utils.IsFileExist(common.RawClusterfile) {
		bytes, err := ioutil.ReadFile(common.RawClusterfile)
		if err != nil {
			return err
		}
		if l.Image.Annotations == nil {
			l.Image.Annotations = make(map[string]string)
		}
		l.Image.Annotations[common.ImageAnnotationForClusterfile] = string(bytes)
	}

	if err := utils.MarshalYamlToFile(filename, l.Image); err != nil {
		return fmt.Errorf("failed to write image yaml:%v", err)
	}

	logger.Info("write image yaml file to %s success !", filename)
	if err := imageUtils.SetImageMetadata(imageUtils.ImageMetadata{
		Name: l.ImageName,
		ID:   l.Image.Spec.ID,
	}); err != nil {
		return fmt.Errorf("failed to set image metadata :%v", err)
	}
	logger.Info("update image %s to image metadata success !", l.ImageName)
	return nil
}

func (l *LocalBuilder) PushToRegistry() error {
	//push image
	err := image.NewImageService().Push(l.ImageName)
	if err != nil {
		return fmt.Errorf("failed to push image :%v", err)
	}
	logger.Info("push image %s to registry success !", l.ImageName)
	return nil
}

func NewLocalBuilder(config *Config) Interface {
	c := new(LocalBuilder)
	c.Config = config
	return c
}

// used in build stage, where the image still has from layer
func getBaseLayersFromImage(image v1.Image) (res []string, err error) {
	if len(image.Spec.Layers) == 0 {
		return nil, fmt.Errorf("no layer found in image %s", image.Name)
	}
	if image.Spec.Layers[0].Value == common.ImageScratch {
		return []string{}, nil
	}

	var layers []v1.Layer
	if image.Spec.Layers[0].Type == common.FROMCOMMAND {
		baseImage, err := imageUtils.GetImage(image.Spec.Layers[0].Value)
		if err != nil {
			return []string{}, err
		}
		if len(baseImage.Spec.Layers) == 0 || baseImage.Spec.Layers[0].Type == common.FROMCOMMAND {
			return []string{}, fmt.Errorf("no layer found in local base image %s, or this base image has base image, which is not allowed", baseImage.Spec.ID)
		}
		layers = append(layers, baseImage.Spec.Layers...)
		// remove the from layer
		//image.Spec.Layers = image.Spec.Layers[1:]
	}
	// TODO the original logic would append current image layers, but I guess there is no need to do that
	//layers = append(layers, image.Spec.Layers...)
	if len(layers) > 128 {
		return []string{}, fmt.Errorf("current layer is exceed 128 layers")
	}

	for _, layer := range layers {
		if layer.Hash != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.Hash))
		}
	}
	return res, nil
}
