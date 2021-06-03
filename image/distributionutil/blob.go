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

	"github.com/docker/distribution"
	"github.com/opencontainers/go-digest"
)

type blobService struct {
	descriptors map[digest.Digest]distribution.Descriptor
}

func (bs *blobService) Get(ctx context.Context, dgst digest.Digest) ([]byte, error) {
	return []byte{}, nil
}

func (bs *blobService) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	if descriptor, ok := bs.descriptors[dgst]; ok {
		return descriptor, nil
	}
	return distribution.Descriptor{}, distribution.ErrBlobUnknown
}

func (bs *blobService) Open(ctx context.Context, dgst digest.Digest) (distribution.ReadSeekCloser, error) {
	return nil, nil
}

func (bs *blobService) Put(ctx context.Context, mediaType string, p []byte) (distribution.Descriptor, error) {
	d := distribution.Descriptor{
		Digest:    digest.FromBytes(p),
		Size:      int64(len(p)),
		MediaType: mediaType,
	}
	bs.descriptors[d.Digest] = d
	return d, nil
}

func (bs *blobService) Create(ctx context.Context, options ...distribution.BlobCreateOption) (distribution.BlobWriter, error) {
	return nil, nil
}

func (bs *blobService) Resume(ctx context.Context, id string) (distribution.BlobWriter, error) {
	return nil, nil
}
