// Copyright © 2022 Alibaba Group Holding Ltd.
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
	"errors"
	"fmt"

	"github.com/sealerio/sealer/build/buildinstruction"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/pkg/image/store"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/maps"
	"github.com/sealerio/sealer/utils/mount"
	"github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"

	"golang.org/x/sync/errgroup"
)

const (
	maxLayerDeep = 128
)

type layerExecutor struct {
	platform        v1.Platform
	baseLayers      []v1.Layer
	layerStore      store.LayerStore
	rootfsMountInfo mount.Service
}

func (l *layerExecutor) Execute(ctx Context, rawLayers []v1.Layer) ([]v1.Layer, error) {
	var (
		execCtx    buildinstruction.ExecContext
		baseLayers = l.baseLayers
	)

	// process middleware file
	err := l.checkMiddleware(ctx.BuildContext)
	if err != nil {
		return []v1.Layer{}, err
	}

	execCtx = buildinstruction.NewExecContext(ctx.BuildContext, ctx.BuildArgs,
		ctx.UseCache, l.layerStore)

	for i := 0; i < len(rawLayers); i++ {
		//we are to set layer id for each new layers.
		layer := &rawLayers[i]
		logrus.Infof("run build layer: %s %s", layer.Type, layer.Value)

		if layer.Type == common.CMDCOMMAND {
			continue
		}

		//run layer instruction exec to get layer id and cache id
		ic := buildinstruction.InstructionContext{
			BaseLayers:   baseLayers,
			CurrentLayer: layer,
			Platform:     l.platform,
		}
		inst, err := buildinstruction.NewInstruction(ic)
		if err != nil {
			return []v1.Layer{}, err
		}
		out, err := inst.Exec(execCtx)
		if err != nil {
			return []v1.Layer{}, err
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
	logrus.Info("exec all build instructs success")

	// process differ of manifests and metadata.
	err = l.checkDiff(rawLayers)
	if err != nil {
		return []v1.Layer{}, err
	}

	upper := l.rootfsMountInfo.GetMountUpper()
	layer, err := l.genNewLayer(common.BaseImageLayerType, common.RootfsLayerValue, upper)
	if err != nil {
		return []v1.Layer{}, err
	}

	if layer.ID != "" {
		baseLayers = append(baseLayers, layer)
	} else {
		logrus.Warn("no rootfs diff content found")
	}

	return baseLayers, nil
}

func (l *layerExecutor) checkMiddleware(buildContext string) error {
	var (
		rootfs      = l.rootfsMountInfo.GetMountTarget()
		middlewares = []Differ{NewMiddlewarePuller(l.platform)}
	)
	logrus.Info("start to check the middleware file")
	eg, _ := errgroup.WithContext(context.Background())
	for _, middleware := range middlewares {
		s := middleware
		eg.Go(func() error {
			err := s.Process(buildContext, rootfs)
			if err != nil {
				return err
			}
			return nil
		})
	}
	return eg.Wait()
}

func (l *layerExecutor) checkDiff(rawLayers []v1.Layer) error {
	var (
		rootfs  = l.rootfsMountInfo.GetMountTarget()
		eg, _   = errgroup.WithContext(context.Background())
		differs = []Differ{NewRegistryDiffer(l.platform), NewMetadataDiffer()}
	)
	mi, err := GetLayerMountInfo(rawLayers)
	if err != nil {
		return err
	}
	defer mi.CleanUp()

	srcPath := mi.GetMountTarget()
	for _, diff := range differs {
		d := diff
		eg.Go(func() error {
			err = d.Process(srcPath, rootfs)
			if err != nil {
				return err
			}
			return nil
		})
	}
	return eg.Wait()
}

func (l *layerExecutor) genNewLayer(layerType, layerValue, filepath string) (v1.Layer, error) {
	imageLayer := v1.Layer{
		Type:  layerType,
		Value: layerValue,
	}

	layerID, err := l.layerStore.RegisterLayerForBuilder(filepath)
	if err != nil {
		return imageLayer, fmt.Errorf("failed to register layer, err: %v", err)
	}

	imageLayer.ID = layerID
	return imageLayer, nil
}

func (l *layerExecutor) Cleanup() error {
	l.rootfsMountInfo.CleanUp()
	return nil
}

func NewLayerExecutor(baseLayers []v1.Layer, platform v1.Platform) (Executor, error) {
	mountInfo, err := GetLayerMountInfo(baseLayers)
	if err != nil {
		return nil, err
	}
	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, err
	}

	return &layerExecutor{
		baseLayers:      baseLayers,
		layerStore:      layerStore,
		rootfsMountInfo: mountInfo,
		platform:        platform,
	}, nil
}

// NewBuildImageByKubefile init image spec by kubefile and check if base image exists ,if not will pull it.
func NewBuildImageByKubefile(kubefileName string, platform v1.Platform) (*v1.Image, []v1.Layer, error) {
	rawImage, err := initImageSpec(kubefileName)
	if err != nil {
		return nil, nil, err
	}

	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, nil, err
	}

	service, err := image.NewImageService()
	if err != nil {
		return nil, nil, err
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
		plats := []*v1.Platform{&platform}
		if err = service.PullIfNotExist(layer0.Value, plats); err != nil {
			return nil, nil, fmt.Errorf("failed to pull baseImage: %v", err)
		}
		baseImage, err = imageStore.GetByName(layer0.Value, &platform)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get base image: %s", err)
		}
	}

	baseLayers := append([]v1.Layer{}, baseImage.Spec.Layers...)
	newLayers := append([]v1.Layer{}, rawImage.Spec.Layers[1:]...)
	if len(baseLayers)+len(newLayers) > maxLayerDeep {
		return nil, nil, errors.New("current number of layers exceeds 128 layers")
	}

	// merge base image cmd and set to raw image as parent.
	rawImage.Spec.ImageConfig.Cmd.Parent = strings.Merge(baseImage.Spec.ImageConfig.Cmd.Parent,
		baseImage.Spec.ImageConfig.Cmd.Current)
	// merge base image args and set to raw image as parent.
	rawImage.Spec.ImageConfig.Args.Parent = maps.Merge(baseImage.Spec.ImageConfig.Args.Parent,
		baseImage.Spec.ImageConfig.Args.Current)

	return rawImage, baseLayers, nil
}
