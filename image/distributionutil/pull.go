package distributionutil

import (
	"context"
	"encoding/json"

	"fmt"
	"path/filepath"
	"sync"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/compress"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/pkg/progress"
	"github.com/justadogistaken/reg/registry"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type Puller interface {
	Pull(context context.Context, named reference.Named) (*v1.Image, error)
}

type ImagePuller struct {
	config   Config
	registry *registry.Registry
}

func (puller *ImagePuller) Pull(context context.Context, named reference.Named) (*v1.Image, error) {
	var (
		errorCh    = make(chan error, 128)
		done       sync.WaitGroup
		layerStore = puller.config.LayerStore
	)

	manifest, err := puller.getRemoteManifest(context, named)
	if err != nil {
		return nil, err
	}

	v1Image, err := puller.getRemoteImageMetadata(context, named, manifest.Config.Digest)
	if err != nil {
		return nil, err
	}
	for _, l := range manifest.Layers {
		done.Add(1)
		go func(layer distribution.Descriptor) {
			defer done.Done()
			roLayer, LayerErr := store.NewROLayer(layer.Digest, layer.Size)
			if LayerErr != nil {
				errorCh <- LayerErr
				return
			}

			LayerErr = puller.downloadLayer(context, named, roLayer)
			if LayerErr != nil {
				errorCh <- LayerErr
				return
			}

			LayerErr = layerStore.RegisterLayerIfNotPresent(roLayer)
			if LayerErr != nil {
				errorCh <- LayerErr
				return
			}
		}(l)
	}
	done.Wait()
	if len(errorCh) > 0 {
		close(errorCh)
		err = fmt.Errorf("failed to pull image %s", named.Raw())
		for chErr := range errorCh {
			err = errors.Wrap(chErr, err.Error())
		}
		return nil, err
	}

	return &v1Image, nil
}

func (puller *ImagePuller) downloadLayer(ctx context.Context, named reference.Named, layer store.Layer) error {
	var (
		layerStore     = puller.config.LayerStore
		progressOut    = puller.config.ProgressOutput
		registryClient = puller.registry
	)

	// check layer existence
	roLayer := layerStore.Get(layer.ID())
	if roLayer != nil {
		progress.Message(progressOut, layer.SimpleID(), "already exists")
		return nil
	}

	layerDownloadReader, err := registryClient.DownloadLayer(ctx, named.Repo(), digest.Digest(layer.ID()))
	if err != nil {
		return err
	}

	progressReader := progress.NewProgressReader(layerDownloadReader, progressOut, layer.Size(), layer.SimpleID(), "pulling")
	err = compress.Decompress(progressReader, filepath.Join(common.DefaultLayerDir, digest.Digest(layer.ID()).Hex()))
	if err != nil {
		progress.Update(progressOut, layer.SimpleID(), err.Error())
		return err
	}

	progress.Update(progressOut, layer.SimpleID(), "pull completed")
	return nil
}

// TODO make a manifest store do this job
func (puller *ImagePuller) getRemoteManifest(context context.Context, named reference.Named) (schema2.Manifest, error) {
	return puller.registry.ManifestV2(context, named.Repo(), named.Tag())
}

// not docker image, get sealer image metadata
func (puller *ImagePuller) getRemoteImageMetadata(context context.Context, named reference.Named, digest digest.Digest) (v1.Image, error) {
	manifestImage, err := puller.registry.DownloadLayer(context, named.Repo(), digest)
	if err != nil {
		return v1.Image{}, err
	}

	decoder := json.NewDecoder(manifestImage)
	img := v1.Image{}
	return img, decoder.Decode(&img)
}

func NewPuller(config Config) (Puller, error) {
	reg, err := fetchRegistryClient(config.AuthInfo)
	if err != nil {
		return nil, err
	}

	return &ImagePuller{
		registry: reg,
		config:   config,
	}, nil
}
