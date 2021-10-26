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

package buildimage

import (
	"fmt"

	"github.com/alibaba/sealer/build/buildkit/buildinstruction"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/pkg/errors"

	"github.com/alibaba/sealer/logger"
)

const (
	maxLayerDeep = 128
)

type Interface interface {
	GenNewLayer(layerType, layerValue, filepath string) (v1.Layer, error)
	SaveBuildImage(name string, layers []v1.Layer) error
	ExecBuild(ctx Context) error
	GetBaseImageName() string
	GetRawImageBaseLayers() []v1.Layer
	GetRawImageNewLayers() []v1.Layer
}

// BuildImage this struct aims to provide image object in build stage
// include handle layers,save build images to system.
type BuildImage struct {
	RawImage   *v1.Image
	BaseLayers []v1.Layer
	NewLayers  []v1.Layer
	// save image and get base image layers
	ImageStore   store.ImageStore
	LayerStore   store.LayerStore
	ImageService image.Service
}

func (b BuildImage) GetBaseImageName() string {
	return b.RawImage.Spec.Layers[0].Value
}
func (b BuildImage) GetRawImageBaseLayers() []v1.Layer {
	return b.BaseLayers
}

func (b BuildImage) GetRawImageNewLayers() []v1.Layer {
	return b.NewLayers
}

func (b BuildImage) ExecBuild(ctx Context) error {
	var (
		execCtx    buildinstruction.ExecContext
		newLayers  = b.NewLayers
		baseLayers = b.BaseLayers
	)

	if ctx.UseCache {
		execCtx = buildinstruction.NewExecContext(ctx.BuildType, ctx.BuildContext, b.ImageService, b.LayerStore)
	} else {
		execCtx = buildinstruction.NewExecContextWithoutCache(ctx.BuildType, ctx.BuildContext, b.LayerStore)
	}

	for i := 0; i < len(newLayers); i++ {
		//we are to set layer id for each new layers.
		layer := &newLayers[i]
		logger.Info("run build layer: %s %s", layer.Type, layer.Value)

		if ctx.BuildType == common.LiteBuild && layer.Type == common.CMDCOMMAND {
			continue
		}

		//run layer instruction exec to get layer id and cache id
		ic := buildinstruction.InstructionContext{
			BaseLayers:   baseLayers,
			CurrentLayer: layer,
		}
		inst, err := buildinstruction.NewInstruction(ic)
		if err != nil {
			return err
		}
		out, err := inst.Exec(execCtx)
		if err != nil {
			return err
		}

		// update current layer cache status for next cache
		if execCtx.ContinueCache {
			execCtx.ParentID = out.ParentID
			execCtx.ContinueCache = out.ContinueCache
		}
		layer.ID = out.LayerID
		if out.LayerID == "" {
			continue
		}

		baseLayers = append(baseLayers, *layer)
	}

	logger.Info("exec all build instructs success !")
	return nil
}

func (b BuildImage) GenNewLayer(layerType, layerValue, filepath string) (v1.Layer, error) {
	imageLayer := v1.Layer{
		Type:  layerType,
		Value: layerValue,
	}

	layerID, err := b.LayerStore.RegisterLayerForBuilder(filepath)
	if err != nil {
		return imageLayer, fmt.Errorf("failed to register layer, err: %v", err)
	}

	imageLayer.ID = layerID
	return imageLayer, nil
}

func (b BuildImage) SaveBuildImage(name string, layers []v1.Layer) error {
	clusterfile, err := b.getRawImageClusterData()
	if err != nil {
		return err
	}

	err = setClusterFileToImage(clusterfile, name, b.RawImage)
	if err != nil {
		return fmt.Errorf("failed to set image metadata, err: %v", err)
	}

	b.RawImage.Spec.Layers = layers

	err = b.updateImageIDAndSaveImage(name)
	if err != nil {
		return fmt.Errorf("failed to save image metadata, err: %v", err)
	}

	logger.Info("update image %s to image metadata success !", name)
	return nil
}

func (b BuildImage) getRawImageClusterData() (string, error) {
	cluster, err := GetRawClusterFile(b.RawImage.Spec.Layers[0].Value, b.NewLayers)
	if err != nil {
		return "", fmt.Errorf("failed to get base image err: %s", err)
	}
	return cluster, nil
}

func (b BuildImage) updateImageIDAndSaveImage(name string) error {
	imageID, err := generateImageID(*b.RawImage)
	if err != nil {
		return err
	}

	b.RawImage.Spec.ID = imageID
	return b.ImageStore.Save(*b.RawImage, name)
}

func NewBuildImage(kubefileName string) (Interface, error) {
	rawImage, err := InitImageSpec(kubefileName)
	if err != nil {
		return nil, err
	}

	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, err
	}

	service, err := image.NewImageService()
	if err != nil {
		return nil, err
	}

	var (
		layer0    = rawImage.Spec.Layers[0]
		baseImage *v1.Image
	)

	// and the layer 0 must be from layer
	if layer0.Value == common.ImageScratch {
		// give an empty image
		baseImage = &v1.Image{}
	} else {
		if err = service.PullIfNotExist(layer0.Value); err != nil {
			return nil, fmt.Errorf("failed to pull baseImage: %v", err)
		}
		baseImage, err = imageStore.GetByName(layer0.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to get base image err: %s", err)
		}
	}

	baseLayers := append([]v1.Layer{}, baseImage.Spec.Layers...)
	newLayers := append([]v1.Layer{}, rawImage.Spec.Layers[1:]...)
	if len(baseLayers)+len(newLayers) > maxLayerDeep {
		return nil, errors.New("current number of layers exceeds 128 layers")
	}

	return &BuildImage{
		RawImage:     rawImage,
		ImageStore:   imageStore,
		ImageService: service,
		LayerStore:   layerStore,
		BaseLayers:   baseLayers,
		NewLayers:    newLayers,
	}, nil
}
