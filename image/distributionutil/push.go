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
	"io"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"

	"os"
	"path/filepath"
	"sync"

	"github.com/alibaba/sealer/utils/compress"
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
}

func (pusher *ImagePusher) Push(ctx context.Context, named reference.Named) error {
	var (
		layerStore          = pusher.config.LayerStore
		done                sync.WaitGroup
		errorCh             = make(chan error, 128)
		layerDescriptorChan chan distribution.Descriptor
	)

	image, err := imageutils.GetImage(named.Raw())
	if err != nil {
		return err
	}

	// to get layer descriptors, for building image manifest
	layerDescriptorChan = make(chan distribution.Descriptor, len(image.Spec.Layers))
	for _, l := range image.Spec.Layers {
		if l.Hash == "" {
			continue
		}

		roLayer, err := store.NewROLayer(l.Hash, 0)
		if err != nil {
			return err
		}
		if layerStore.Get(roLayer.ID()) == nil {
			return fmt.Errorf("failed to put image %s, layer %s not exists locally", named.Raw(), roLayer.SimpleID())
		}

		done.Add(1)
		go func(layer store.Layer) {
			defer done.Done()

			layerDescriptor, layerErr := pusher.uploadLayer(ctx, named, layer)
			if layerErr != nil {
				errorCh <- layerErr
				return
			}
			layerDescriptorChan <- layerDescriptor
		}(roLayer)
	}
	done.Wait()
	if len(errorCh) > 0 {
		close(errorCh)
		err = fmt.Errorf("failed to push image %s", named.Raw())
		for chErr := range errorCh {
			err = errors.Wrap(chErr, err.Error())
		}
		return err
	}

	var layerDescriptors []distribution.Descriptor
	close(layerDescriptorChan)
	for descriptor := range layerDescriptorChan {
		layerDescriptors = append(layerDescriptors, descriptor)
	}

	// push sealer image metadata to registry
	configJSON, err := pusher.putManifestConfig(ctx, named, *image)
	if err != nil {
		return err
	}

	return pusher.putManifest(ctx, configJSON, named, layerDescriptors)
}

func (pusher *ImagePusher) uploadLayer(ctx context.Context, named reference.Named, layer store.Layer) (distribution.Descriptor, error) {
	var (
		file            *os.File
		repo            = pusher.repository
		progressChanOut = pusher.config.ProgressOutput
		layerIDDigest   = digest.Digest(layer.ID())
	)

	bs := repo.Blobs(ctx)
	// check if layer exists remotely.
	remoteLayerDescriptor, err := bs.Stat(ctx, layerIDDigest)
	if err == nil {
		progress.Message(progressChanOut, layer.SimpleID(), "already exists")
		return remoteLayerDescriptor, nil
	}

	// pack layer files into tar.gz
	progress.Update(progressChanOut, layer.SimpleID(), "preparing")
	if file, err = compress.RootDirNotIncluded(nil, filepath.Join(common.DefaultLayerDir, layerIDDigest.Hex())); err != nil {
		return distribution.Descriptor{}, err
	}
	defer utils.CleanFile(file)

	_, err = file.Seek(0, 0)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	fi, err := file.Stat()
	if err != nil {
		return distribution.Descriptor{}, err
	}

	progressReader := progress.NewProgressReader(file, progressChanOut, fi.Size(), layer.SimpleID(), "pushing")
	defer progressReader.Close()

	layerUploader, err := bs.Create(ctx)
	if err != nil {
		return distribution.Descriptor{}, err
	}
	defer layerUploader.Close()

	digester := digest.Canonical.Digester()
	tee := io.TeeReader(progressReader, digester.Hash())
	_, err = layerUploader.ReadFrom(tee)
	if err != nil {
		return distribution.Descriptor{}, fmt.Errorf("failed to upload layer %s, err: %s", layer.ID(), err)
	}
	if digester.Digest() != layerIDDigest {
		return distribution.Descriptor{}, fmt.Errorf("layer hash changed, which means the layer filesystem may changed, current: %s, original: %s", digester.Digest(), layerIDDigest)
	}
	if _, err = layerUploader.Commit(ctx, distribution.Descriptor{Digest: layerIDDigest}); err != nil {
		return distribution.Descriptor{}, fmt.Errorf("failed to commit layer to registry, err: %s", err)
	}

	progress.Update(progressChanOut, layer.SimpleID(), "push completed")
	return buildBlobs(layerIDDigest, fi.Size(), schema2.MediaTypeLayer), nil
}

func (pusher *ImagePusher) putManifest(ctx context.Context, configJSON []byte, named reference.Named, layerDescriptors []distribution.Descriptor) error {
	var (
		bs   = &blobService{descriptors: map[digest.Digest]distribution.Descriptor{}}
		repo = pusher.repository
	)
	manifestBuilder := schema2.NewManifestBuilder(
		bs,
		//TODO use schema2.MediaTypeImageConfig by default
		//plan to support more types to support more registry
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

	return &ImagePusher{
		repository: repo,
		config:     config,
	}, nil
}
