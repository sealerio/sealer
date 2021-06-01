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
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"

	"os"
	"path/filepath"
	"sync"

	"github.com/alibaba/sealer/registry"
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
	config   Config
	registry *registry.Registry // registrysdk
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
		registryCli     = pusher.registry
		progressChanOut = pusher.config.ProgressOutput
		layerID         = digest.Digest(layer.ID())
	)

	// check if layer exists remotely.
	remoteLayer, err := registryCli.LayerMetadata(named.Repo(), layerID)
	if err == nil {
		progress.Message(progressChanOut, layer.SimpleID(), "already exists")
		return remoteLayer, nil
	}

	// pack layer files into tar.gz
	progress.Update(progressChanOut, layer.SimpleID(), "preparing")
	if file, err = compress.RootDirNotIncluded(nil, filepath.Join(common.DefaultLayerDir, layerID.Hex())); err != nil {
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

	err = registryCli.UploadLayer(ctx, named.Repo(), layerID, progressReader)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	progress.Update(progressChanOut, layer.SimpleID(), "push completed")
	return buildBlobs(layerID, fi.Size(), schema2.MediaTypeLayer), nil
}

func (pusher *ImagePusher) putManifest(ctx context.Context, configJSON []byte, named reference.Named, layerDescriptors []distribution.Descriptor) error {
	bs := &blobService{descriptors: map[digest.Digest]distribution.Descriptor{}}
	manifestBuilder := schema2.NewManifestBuilder(
		bs,
		schema2.MediaTypeManifest,
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

	return pusher.registry.PutManifest(ctx, named.Repo(), named.Tag(), manifest)
}

func (pusher *ImagePusher) putManifestConfig(ctx context.Context, named reference.Named, image v1.Image) ([]byte, error) {
	configJSON, err := json.Marshal(image)
	if err != nil {
		return nil, err
	}

	dig := digest.FromBytes(configJSON)
	err = pusher.registry.UploadLayer(ctx, named.Repo(), dig, bytes.NewReader(configJSON))
	return configJSON, err
}

func buildBlobs(dig digest.Digest, size int64, mediaType string) distribution.Descriptor {
	return distribution.Descriptor{
		Digest:    dig,
		Size:      size,
		MediaType: mediaType,
	}
}

func NewPusher(config Config) (Pusher, error) {
	regCli, err := fetchRegistryClient(config.AuthInfo)
	if err != nil {
		return nil, err
	}

	return &ImagePusher{
		registry: regCli,
		config:   config,
	}, nil
}
