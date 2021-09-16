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
	"fmt"
	"github.com/alibaba/sealer/pkg/image/reference"
	store2 "github.com/alibaba/sealer/pkg/image/store"
	"io"

	"golang.org/x/sync/errgroup"

	v1 "github.com/alibaba/sealer/types/api/v1"

	"sync"

	"github.com/alibaba/sealer/utils/archive"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/pkg/progress"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type Pusher interface {
	Push(ctx context.Context, named reference.Named) error
}

type ImagePusher struct {
	config     Config
	repository distribution.Repository
	imageStore store2.ImageStore
}

func (pusher *ImagePusher) Push(ctx context.Context, named reference.Named) error {
	var (
		layerStore   = pusher.config.LayerStore
		pushedLayers = map[string]distribution.Descriptor{}
		pushMux      sync.Mutex
		eg           *errgroup.Group
	)

	image, err := pusher.imageStore.GetByName(named.Raw())
	if err != nil {
		return err
	}

	eg, _ = errgroup.WithContext(context.Background())
	for _, l := range image.Spec.Layers {
		if l.ID == "" {
			continue
		}
		err := l.ID.Validate()
		if err != nil {
			return fmt.Errorf("layer hash %s validate failed, err: %s", l.ID, err)
		}

		// this scope value, safe to pass into eg.Go
		roLayer := layerStore.Get(store2.LayerID(l.ID))
		if roLayer == nil {
			return fmt.Errorf("failed to put image %s, layer %s not exists locally", named.Raw(), l.ID.String())
		}

		eg.Go(func() error {
			layerDescriptor, layerErr := pusher.uploadLayer(ctx, roLayer)
			if layerErr != nil {
				return layerErr
			}

			pushMux.Lock()
			pushedLayers[roLayer.ID().String()] = layerDescriptor
			pushMux.Unlock()
			// add distribution digest metadata to disk
			return layerStore.AddDistributionMetadata(roLayer.ID(), named, layerDescriptor.Digest)
		})
	}
	err = eg.Wait()
	if err != nil {
		return fmt.Errorf("failed to push layers of %s, err: %s", named.Raw(), err)
	}

	// for making descriptors have same order with image layers
	// descriptor and image yaml are both saved in registry
	// but they are different, layer digest in layer yaml is layerid.
	// And digest in descriptor indicate the hash of layer content.
	var layerDescriptors []distribution.Descriptor
	for _, l := range image.Spec.Layers {
		if l.ID == "" {
			continue
		}
		// l.Hash.String() is same as layer.ID().String() above
		layerDescriptor, ok := pushedLayers[l.ID.String()]
		if !ok {
			continue
		}
		layerDescriptors = append(layerDescriptors, layerDescriptor)
	}
	if len(layerDescriptors) != len(pushedLayers) {
		return errors.New("failed to push image, the number of layerDescriptors and pushedLayers mismatch")
	}
	// push sealer image metadata to registry
	configJSON, err := pusher.putManifestConfig(ctx, named, *image)
	if err != nil {
		return err
	}

	return pusher.putManifest(ctx, configJSON, named, layerDescriptors)
}

func (pusher *ImagePusher) uploadLayer(ctx context.Context, roLayer store2.Layer) (distribution.Descriptor, error) {
	var (
		err                      error
		layerContentStream       io.ReadCloser
		repo                     = pusher.repository
		progressChanOut          = pusher.config.ProgressOutput
		layerDistributionDigests = roLayer.DistributionMetadata()
	)

	bs := repo.Blobs(ctx)
	// if layerDistributionDigests is empty, we take the layer inexistence in the registry
	// check all candidates
	if len(layerDistributionDigests) > 0 {
		// check if layer exists remotely.
		for _, cand := range layerDistributionDigests {
			remoteLayerDescriptor, err := bs.Stat(ctx, cand)
			if err == nil {
				progress.Message(progressChanOut, roLayer.SimpleID(), "already exists")
				return remoteLayerDescriptor, nil
			}
		}
	}

	// pack layer files into tar.gz
	progress.Update(progressChanOut, roLayer.SimpleID(), "preparing")
	layerContentStream, err = roLayer.TarStream()
	if err != nil {
		return distribution.Descriptor{}, errors.Errorf("failed to get tar stream for layer %s, err: %s", roLayer.ID(), err)
	}
	//progress.NewProgressReader will close layerContentStream
	progressReader := progress.NewProgressReader(layerContentStream, progressChanOut, roLayer.Size(), roLayer.SimpleID(), "pushing")
	uploadStream, _ := archive.GzipCompress(progressReader)
	defer func() {
		layerContentStream.Close()
		uploadStream.Close()
	}()

	layerUploader, err := bs.Create(ctx)
	if err != nil {
		progress.Update(progressChanOut, roLayer.SimpleID(), "push failed")
		return distribution.Descriptor{}, err
	}
	defer layerUploader.Close()

	// calculate hash of layer content stream
	digester := digest.Canonical.Digester()
	tee := io.TeeReader(uploadStream, digester.Hash())
	realSize, err := layerUploader.ReadFrom(tee)
	if err != nil {
		return distribution.Descriptor{}, fmt.Errorf("failed to upload layer %s, err: %s", roLayer.ID(), err)
	}

	layerContentDigest := digester.Digest()
	if _, err = layerUploader.Commit(ctx, distribution.Descriptor{Digest: layerContentDigest}); err != nil {
		return distribution.Descriptor{}, fmt.Errorf("failed to commit layer to registry, err: %s", err)
	}

	progress.Update(progressChanOut, roLayer.SimpleID(), "push completed")
	return buildBlobs(layerContentDigest, realSize, roLayer.MediaType()), nil
}

func (pusher *ImagePusher) putManifest(ctx context.Context, configJSON []byte, named reference.Named, layerDescriptors []distribution.Descriptor) error {
	var (
		bs   = &blobService{descriptors: map[digest.Digest]distribution.Descriptor{}}
		repo = pusher.repository
	)
	manifestBuilder := schema2.NewManifestBuilder(
		bs,
		// use schema2.MediaTypeImageConfig by default
		//TODO plan to support more types to support more registry
		schema2.MediaTypeImageConfig,
		configJSON)

	for _, d := range layerDescriptors {
		err := manifestBuilder.AppendReference(d)
		if err != nil {
			return err
		}
	}

	manifest, err := manifestBuilder.Build(ctx)
	if err != nil {
		return err
	}

	ms, err := repo.Manifests(ctx)
	if err != nil {
		return err
	}

	putOptions := []distribution.ManifestServiceOption{distribution.WithTag(named.Tag())}
	_, err = ms.Put(ctx, manifest, putOptions...)
	return err
}

func (pusher *ImagePusher) putManifestConfig(ctx context.Context, named reference.Named, image v1.Image) ([]byte, error) {
	repo := pusher.repository
	configJSON, err := json.Marshal(image)
	if err != nil {
		return nil, err
	}

	bs := repo.Blobs(ctx)
	_, err = bs.Put(ctx, schema2.MediaTypeImageConfig, configJSON)
	if err != nil {
		return nil, err
	}

	return configJSON, err
}

func buildBlobs(dig digest.Digest, size int64, mediaType string) distribution.Descriptor {
	return distribution.Descriptor{
		Digest:    dig,
		Size:      size,
		MediaType: mediaType,
	}
}

func NewPusher(named reference.Named, config Config) (Pusher, error) {
	repo, err := NewV2Repository(named, "push", "pull")
	if err != nil {
		return nil, err
	}

	is, err := store2.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return &ImagePusher{
		repository: repo,
		config:     config,
		imageStore: is,
	}, nil
}
