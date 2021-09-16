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
	"github.com/alibaba/sealer/pkg/image/reference"
	"io"

	"github.com/opencontainers/go-digest"
)

type LayerStore interface {
	Get(id LayerID) Layer
	RegisterLayerIfNotPresent(layer Layer) error
	RegisterLayerForBuilder(diffPath string) (digest.Digest, error)
	Delete(id LayerID) error
	DisassembleTar(layerID digest.Digest, streamReader io.ReadCloser) error
	AddDistributionMetadata(layerID LayerID, named reference.Named, descriptorDigest digest.Digest) error
}

type Layer interface {
	ID() LayerID
	TarStream() (io.ReadCloser, error)
	SimpleID() string
	Size() int64
	MediaType() string
	DistributionMetadata() map[string]digest.Digest
	SetSize(size int64)
}
