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
	"context"
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/pkg/runtime"
	"golang.org/x/sync/errgroup"

	"github.com/alibaba/sealer/build/buildkit/buildinstruction"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/pkg/errors"
)

const (
	maxLayerDeep = 128
)

// BuildImage this struct aims to provide image object in build stage
// include handle layers,save build images to system.
type BuildImage struct {
	BuildType       string
	RawImage        *v1.Image
	BaseLayers      []v1.Layer
	NewLayers       []v1.Layer
	ImageStore      store.ImageStore
	LayerStore      store.LayerStore
	ImageService    image.Service
	RootfsMountInfo *buildinstruction.MountTarget
}

func (b BuildImage) ExecBuild(ctx Context) error {
	var (
		execCtx    buildinstruction.ExecContext
		newLayers  = b.NewLayers
		baseLayers = b.BaseLayers
	)

	if len(ctx.BuildArgs) != 0 {
		for k, v := range ctx.BuildArgs {
			b.RawImage.Spec.ImageConfig.Args[k] = v
		}
	}

	if ctx.UseCache {
		execCtx = buildinstruction.NewExecContext(b.BuildType, ctx.BuildContext,
			b.RawImage.Spec.ImageConfig.Args,
			b.ImageService, b.LayerStore)
	} else {
		execCtx = buildinstruction.NewExecContextWithoutCache(b.BuildType,
			ctx.BuildContext,
			b.RawImage.Spec.ImageConfig.Args,
			b.LayerStore)
	}
	for i := 0; i < len(newLayers); i++ {
		//we are to set layer id for each new layers.
		layer := &newLayers[i]
		logger.Info("run build layer: %s %s", layer.Type, layer.Value)

		if b.BuildType == common.LiteBuild && layer.Type == common.CMDCOMMAND {
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

func (b BuildImage) genNewLayer(layerType, layerValue, filepath string) (v1.Layer, error) {
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

func (b BuildImage) checkDiff() error {
	mi, err := GetLayerMountInfo(b.NewLayers, b.BuildType)
	if err != nil {
		return err
	}
	defer mi.CleanUp()

	differs := []Differ{NewRegistryDiffer(), NewMetadataDiffer()}
	eg, _ := errgroup.WithContext(context.Background())

	for _, diff := range differs {
		d := diff
		eg.Go(func() error {
			err = d.Process(*mi, *b.RootfsMountInfo)
			if err != nil {
				return err
			}
			return nil
		})
	}
	return eg.Wait()
}

func (b BuildImage) SaveBuildImage(name string, opts SaveOpts) error {
	err := b.checkDiff()
	if err != nil {
		return err
	}

	layers, err := b.collectLayers(opts)
	if err != nil {
		return err
	}

	b.RawImage.Spec.Layers = layers
	err = b.save(name)
	if err != nil {
		return fmt.Errorf("failed to save image metadata, err: %v", err)
	}

	logger.Info("update image %s to image metadata success !", name)
	return nil
}

func (b BuildImage) collectLayers(opts SaveOpts) ([]v1.Layer, error) {
	var layers []v1.Layer

	if opts.WithoutBase {
		layers = b.NewLayers
	} else {
		layers = append(b.BaseLayers, b.NewLayers...)
	}

	layer, err := b.collectRootfsDiff()
	if err != nil {
		return nil, err
	}
	if layer.ID == "" {
		logger.Warn("no rootfs diff content found")
		return layers, nil
	}
	layers = append(layers, layer)
	return layers, nil
}

func (b BuildImage) collectRootfsDiff() (v1.Layer, error) {
	var layer v1.Layer
	upper := b.RootfsMountInfo.GetMountUpper()

	layer, err := b.genNewLayer(common.BaseImageLayerType, common.RootfsLayerValue, upper)
	if err != nil {
		return layer, fmt.Errorf("failed to register layer, err: %v", err)
	}

	return layer, nil
}

func (b BuildImage) save(name string) error {
	mi, err := GetLayerMountInfo(b.RawImage.Spec.Layers, b.BuildType)
	if err != nil {
		return err
	}
	defer mi.CleanUp()

	rootfsPath := mi.GetMountTarget()
	b.RawImage.Name = name
	is := []ImageSetter{NewAnnotationSetter(rootfsPath), NewPlatformSetter(rootfsPath)}
	for _, s := range is {
		if err = s.Set(b.RawImage); err != nil {
			return err
		}
	}
	imageID, err := generateImageID(*b.RawImage)
	if err != nil {
		return err
	}
	b.RawImage.Spec.ID = imageID

	return b.ImageStore.Save(*b.RawImage, name)
}

func (b BuildImage) Cleanup() error {
	b.RootfsMountInfo.CleanUp()
	return nil
}

func NewBuildImage(kubefileName string, buildType string) (Interface, error) {
	rawImage, err := InitImageSpec(kubefileName)
	if err != nil {
		return nil, err
	}

	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, err
	}

	imageStore, err := store.NewDefaultImageStore()
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

	mountInfo, err := GetLayerMountInfo(baseLayers, buildType)
	if err != nil {
		return nil, err
	}

	return &BuildImage{
		BuildType:       buildType,
		RawImage:        rawImage,
		ImageStore:      imageStore,
		LayerStore:      layerStore,
		ImageService:    service,
		BaseLayers:      baseLayers,
		NewLayers:       newLayers,
		RootfsMountInfo: mountInfo,
	}, nil
}

type annotation struct {
	source string
}

func (a annotation) Set(ima *v1.Image) error {
	return a.setClusterFile(ima)
}

func (a annotation) setClusterFile(ima *v1.Image) error {
	cluster, err := LoadClusterFile(filepath.Join(a.source, "etc", common.DefaultClusterFileName))
	if err != nil {
		return fmt.Errorf("failed to load clusterfile, err: %v", err)
	}
	cluster.Spec.Image = ima.Name
	err = setClusterFileToImage(cluster, ima)
	if err != nil {
		return fmt.Errorf("failed to set image metadata, err: %v", err)
	}
	return nil
}

func NewAnnotationSetter(rootfs string) ImageSetter {
	return annotation{
		source: rootfs,
	}
}

type platform struct {
	source string
}

func (p platform) Set(ima *v1.Image) error {
	plat := runtime.GetCloudImagePlatform(p.source)
	ima.Spec.Platform = v1.Platform{
		Architecture: plat.Architecture,
		OS:           plat.OS,
		OSVersion:    plat.OSVersion,
		Variant:      plat.Variant,
	}
	return nil
}
func NewPlatformSetter(rootfs string) ImageSetter {
	return platform{
		source: rootfs,
	}
}
