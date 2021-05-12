package distributionutil

import (
	"context"
	"encoding/json"
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
	"path/filepath"
	"sync"
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
		errorCh    = make(chan error)
		done       sync.WaitGroup
		layerStore = puller.config.LayerStore
	)

	manifest, err := puller.getRemoteManifest(context, named)
	if err != nil {
		return nil, err
	}

	v1Image, err := puller.getRemoteImage(context, named, manifest.Config.Digest)
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
		return nil, errors.Wrap(<-errorCh, "failed to pull image")
	}
	return &v1Image, nil
}

func (puller *ImagePuller) downloadLayer(ctx context.Context, named reference.Named, layer store.Layer) error {
	var (
		layerStore     = puller.config.LayerStore
		progressOut    = puller.config.ProgressOutput
		registryClient = puller.registry
	)

	roLayer := layerStore.Get(layer.ID())
	if roLayer == nil {
		progress.Message(progressOut, layer.SimpleID(), "already exists")
		return nil
	}

	layerDownloadReader, err := registryClient.DownloadLayer(ctx, named.Repo(), digest.Digest(layer.ID()))
	if err != nil {
		return err
	}

	progressReader := progress.NewProgressReader(layerDownloadReader, progressOut, layer.Size(), layer.SimpleID(), "pulling")
	return compress.Decompress(progressReader, filepath.Join(common.DefaultLayerDir, digest.Digest(layer.ID()).Hex()))
}

// TODO make a manifest store do this job
func (puller *ImagePuller) getRemoteManifest(context context.Context, named reference.Named) (schema2.Manifest, error) {
	return puller.registry.ManifestV2(context, named.Repo(), named.Tag())
}

func (puller *ImagePuller) getRemoteImage(context context.Context, named reference.Named, digest digest.Digest) (v1.Image, error) {
	manifestImage, err := puller.registry.DownloadLayer(context, named.Repo(), digest)
	if err != nil {
		return v1.Image{}, err
	}

	decoder := json.NewDecoder(manifestImage)
	img := v1.Image{}
	return img, decoder.Decode(&img)
}

func NewPuller(config Config) (Puller, error) {
	newImagePuller := &ImagePuller{config: config}
	reg, err := fetchRegistryClient(newImagePuller.config.AuthInfo)
	if err != nil {
		return nil, err
	}

	newImagePuller.registry = reg
	return newImagePuller, nil
}
