package buildimage

import (
	"context"
	"errors"
	"fmt"
	"github.com/alibaba/sealer/build/buildkit/buildinstruction"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"golang.org/x/sync/errgroup"
)

const (
	maxLayerDeep = 128
)

type layerExecutor struct {
	buildType       string
	baseLayers      []v1.Layer
	layerStore      store.LayerStore
	rootfsMountInfo *buildinstruction.MountTarget
}

func (l layerExecutor) Execute(ctx Context, rawLayers []v1.Layer) ([]v1.Layer, error) {
	var (
		execCtx    buildinstruction.ExecContext
		baseLayers = l.baseLayers
	)

	// process middleware file
	err := l.checkMiddleware(ctx.BuildContext)
	if err != nil {
		return []v1.Layer{}, err
	}

	execCtx = buildinstruction.NewExecContext(l.buildType, ctx.BuildContext,
		ctx.BuildArgs,
		ctx.UseCache, l.layerStore)

	for i := 0; i < len(rawLayers); i++ {
		//we are to set layer id for each new layers.
		layer := &rawLayers[i]
		logger.Info("run build layer: %s %s", layer.Type, layer.Value)

		if l.buildType == common.LiteBuild && layer.Type == common.CMDCOMMAND {
			continue
		}

		//run layer instruction exec to get layer id and cache id
		ic := buildinstruction.InstructionContext{
			BaseLayers:   baseLayers,
			CurrentLayer: layer,
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
	logger.Info("exec all build instructs success")

	// process differ of manifests and metadata.
	err = l.checkDiff(rawLayers)
	if err != nil {
		return []v1.Layer{}, err
	}

	err = l.collectLayers()
	if err != nil {
		return []v1.Layer{}, err
	}

	return baseLayers, nil
}

func (l layerExecutor) checkMiddleware(buildContext string) error {
	var (
		rootfs      = l.rootfsMountInfo.GetMountTarget()
		middlewares = []Middleware{NewMiddlewarePuller()}
	)
	logger.Info("start to check the middleware file")
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

func (l layerExecutor) checkDiff(rawLayers []v1.Layer) error {
	mi, err := GetLayerMountInfo(rawLayers, l.buildType)
	if err != nil {
		return err
	}
	defer mi.CleanUp()

	differs := []Differ{NewRegistryDiffer(), NewMetadataDiffer()}
	eg, _ := errgroup.WithContext(context.Background())

	for _, diff := range differs {
		d := diff
		eg.Go(func() error {
			err = d.Process(*mi, *l.rootfsMountInfo)
			if err != nil {
				return err
			}
			return nil
		})
	}
	return eg.Wait()
}

func (l layerExecutor) collectLayers() error {
	upper := l.rootfsMountInfo.GetMountUpper()
	layer, err := l.genNewLayer(common.BaseImageLayerType, common.RootfsLayerValue, upper)
	if err != nil {
		return fmt.Errorf("failed to register layer, err: %v", err)
	}

	if layer.ID != "" {
		l.baseLayers = append(l.baseLayers, layer)
	} else {
		logger.Warn("no rootfs diff content found")
	}
	return nil
}

func (l layerExecutor) genNewLayer(layerType, layerValue, filepath string) (v1.Layer, error) {
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

func (l layerExecutor) Cleanup() error {
	l.rootfsMountInfo.CleanUp()
	return nil
}

func NewLayerExecutor(baseLayers []v1.Layer, buildType string) (Executor, error) {
	mountInfo, err := GetLayerMountInfo(baseLayers, buildType)
	if err != nil {
		return nil, err
	}
	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, err
	}

	return layerExecutor{
		buildType:       buildType,
		baseLayers:      baseLayers,
		layerStore:      layerStore,
		rootfsMountInfo: mountInfo,
	}, nil
}

// NewBuildImageByKubefile init image spec by kubefile and check if base image exists ,if not will pull it.
func NewBuildImageByKubefile(kubefileName string) (*v1.Image, []v1.Layer, error) {
	rawImage, err := InitImageSpec(kubefileName)
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
		if err = service.PullIfNotExist(layer0.Value); err != nil {
			return nil, nil, fmt.Errorf("failed to pull baseImage: %v", err)
		}
		baseImage, err = imageStore.GetByName(layer0.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get base image err: %s", err)
		}
	}

	baseLayers := append([]v1.Layer{}, baseImage.Spec.Layers...)
	newLayers := append([]v1.Layer{}, rawImage.Spec.Layers[1:]...)
	if len(baseLayers)+len(newLayers) > maxLayerDeep {
		return nil, nil, errors.New("current number of layers exceeds 128 layers")
	}

	// merge base image cmd and set to raw image as parent.
	rawImage.Spec.ImageConfig.Cmd.Parent = utils.MergeSlice(baseImage.Spec.ImageConfig.Cmd.Parent,
		baseImage.Spec.ImageConfig.Cmd.Current)
	// merge base image args and set to raw image as parent.
	rawImage.Spec.ImageConfig.Args.Parent = utils.MergeMap(baseImage.Spec.ImageConfig.Args.Parent,
		baseImage.Spec.ImageConfig.Args.Current)

	return rawImage, baseLayers, nil
}
