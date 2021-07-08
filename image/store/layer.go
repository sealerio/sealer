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

package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils/archive"

	"github.com/docker/distribution/manifest/schema2"

	"github.com/opencontainers/go-digest"
)

type LayerID digest.Digest

func (li LayerID) String() string {
	return string(li)
}

func (li LayerID) ToDigest() digest.Digest {
	return digest.Digest(li)
}

func (li LayerID) Validate() error {
	return li.ToDigest().Validate()
}

type ROLayer struct {
	id                   LayerID
	size                 int64
	distributionMetadata map[string]digest.Digest
}

func (rl *ROLayer) ID() LayerID {
	return rl.id
}

func (rl *ROLayer) SimpleID() string {
	return digest.Digest(rl.ID()).Hex()[0:12]
}

func (rl *ROLayer) TarStream() (io.ReadCloser, error) {
	layerBackend, err := NewFSStoreBackend()
	if err != nil {
		return nil, err
	}

	var (
		tarDataPath   = filepath.Join(layerBackend.LayerDBDir(digest.Digest(rl.ID())), tarDataGZ)
		layerDataPath = layerBackend.LayerDataDir(rl.ID().ToDigest())
	)
	_, err = os.Stat(tarDataPath)
	// tar-data.json.gz does not exist
	// at the pull stage, the file won't exist
	// so we tar the layer dir.
	if err != nil {
		logger.Debug("failed to find %s for layer %s, use tar, err: %s", tarDataGZ, rl.ID(), err)
		tarReader, tarErr := archive.TarWithoutRootDir(layerDataPath)
		if tarErr != nil {
			return nil, fmt.Errorf("failed to tar layer %s, err: %s", rl.ID(), tarErr)
		}
		return tarReader, nil
	}

	pr, pw := io.Pipe()
	go func() {
		err := layerBackend.assembleTar(rl.ID(), pw)
		if err != nil {
			_ = pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
	}()

	return pr, nil
}

func (rl *ROLayer) Size() int64 {
	return rl.size
}

func (rl *ROLayer) SetSize(size int64) {
	rl.size = size
}

func (rl *ROLayer) MediaType() string {
	return schema2.MediaTypeLayer
}

func (rl *ROLayer) DistributionMetadata() map[string]digest.Digest {
	return rl.distributionMetadata
}

func NewROLayer(LayerDigest digest.Digest, size int64, distributionMetadata map[string]digest.Digest) (*ROLayer, error) {
	err := LayerDigest.Validate()
	if err != nil {
		return nil, err
	}
	if distributionMetadata == nil {
		distributionMetadata = map[string]digest.Digest{}
	}
	return &ROLayer{
		id:                   LayerID(LayerDigest),
		size:                 size,
		distributionMetadata: distributionMetadata,
	}, nil
}
