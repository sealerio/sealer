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

package distributionutil

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/alibaba/sealer/utils/archive"

	"fmt"
	"sync"

	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/pkg/progress"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type Puller interface {
	Pull(context context.Context, named reference.Named) (*v1.Image, error)
}

type ImagePuller struct {
	config     Config
	repository distribution.Repository
}

func (puller *ImagePuller) Pull(context context.Context, named reference.Named) (*v1.Image, error) {
	var (
		errorCh    = make(chan error, 128)
		done       sync.WaitGroup
		layerStore = puller.config.LayerStore
		layers     = []v1.Layer{}
	)

	manifest, err := puller.getRemoteManifest(context, named)
	if err != nil {
		return nil, err
	}

	v1Image, err := puller.getRemoteImageMetadata(context, named, manifest.Config.Digest)
	if err != nil {
		return nil, err
	}
	for _, l := range v1Image.Spec.Layers {
		if l.ID == "" {
			continue
		}
		layers = append(layers, l)
	}
	// number of non-empty layer and layer in distribution should be equal
	if len(layers) != len(manifest.Layers) {
		return nil, fmt.Errorf("the number layerIDs %d and LayerDescriptor %d are mismatch", len(layers), len(manifest.Layers))
	}

	for i, l := range manifest.Layers {
		done.Add(1)
		// we take hash of layer as real layer id,  hash of descriptor is just
		// a identifier for remote data
		go func(descriptor distribution.Descriptor, layer v1.Layer) {
			defer done.Done()
			// roLayer now does not exist, new one
			// descriptor.Size is temp size for this layer
			// real size will be set within downloadLayer
			roLayer, LayerErr := store.NewROLayer(layer.ID,
				descriptor.Size,
				map[string]digest.Digest{named.Domain() + "/" + named.Repo(): descriptor.Digest})
			if LayerErr != nil {
				errorCh <- LayerErr
				return
			}

			LayerErr = puller.downloadLayer(context, named, roLayer, descriptor)
			if LayerErr != nil {
				errorCh <- LayerErr
				return
			}

			LayerErr = layerStore.RegisterLayerIfNotPresent(roLayer)
			if LayerErr != nil {
				errorCh <- LayerErr
				return
			}
		}(l, layers[i])
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

func (puller *ImagePuller) downloadLayer(ctx context.Context, named reference.Named, layer store.Layer, descriptor distribution.Descriptor) error {
	var (
		layerStore  = puller.config.LayerStore
		progressOut = puller.config.ProgressOutput
		repo        = puller.repository
		sfs         = store.NewDefaultLayerStorage()
	)
	// descriptor is remote layer data, but its hash may not be the layer id, so we
	// use layer.ID(hash of layer from v1.Image) to check layer existence.
	roLayer := layerStore.Get(layer.ID())
	if roLayer != nil {
		progress.Message(progressOut, layer.SimpleID(), "already exists")
		return nil
	}

	bs := repo.Blobs(ctx)
	layerReader, err := bs.Open(ctx, descriptor.Digest)
	if err != nil {
		return err
	}
	defer layerReader.Close()

	digester := digest.Canonical.Digester()
	LayerDownloadReader := ioutil.NopCloser(io.TeeReader(layerReader, digester.Hash()))
	progressReader := progress.NewProgressReader(LayerDownloadReader, progressOut, descriptor.Size, layer.SimpleID(), "pulling")
	size, err := archive.Decompress(progressReader, sfs.LayerDataDir(digest.Digest(layer.ID())), archive.Options{Compress: true})
	if err != nil {
		progress.Update(progressOut, layer.SimpleID(), err.Error())
		return err
	}
	// update rolayer size for storing the info under layerdb
	layer.SetSize(size)
	if digester.Digest() != descriptor.Digest {
		return fmt.Errorf("digest verified failed for %s", layer.ID())
	}
	progress.Update(progressOut, layer.SimpleID(), "pull completed")
	return nil
}

// TODO make a manifest store do this job
func (puller *ImagePuller) getRemoteManifest(context context.Context, named reference.Named) (schema2.Manifest, error) {
	repo := puller.repository
	ms, err := repo.Manifests(context)
	if err != nil {
		return schema2.Manifest{}, err
	}

	manifest, err := ms.Get(context, "", distribution.WithTagOption{Tag: named.Tag()})
	if err != nil {
		return schema2.Manifest{}, err
	}

	_, ok := manifest.(*schema2.DeserializedManifest)
	if !ok {
		return schema2.Manifest{}, fmt.Errorf("failed to parse manifest %s to DeserializedManifest", named.RepoTag())
	}
	return manifest.(*schema2.DeserializedManifest).Manifest, nil
}

// not docker image, get sealer image metadata
func (puller *ImagePuller) getRemoteImageMetadata(context context.Context, named reference.Named, digest digest.Digest) (v1.Image, error) {
	repo := puller.repository
	bs := repo.Blobs(context)
	manifestImageBytes, err := bs.Get(context, digest)
	if err != nil {
		return v1.Image{}, err
	}

	img := v1.Image{}
	return img, json.Unmarshal(manifestImageBytes, &img)
}

func NewPuller(named reference.Named, config Config) (Puller, error) {
	repo, err := NewV2Repository(named, "pull")
	if err != nil {
		return nil, err
	}

	return &ImagePuller{
		repository: repo,
		config:     config,
	}, nil
}
