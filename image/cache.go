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

package image

import (
	"fmt"

	"github.com/alibaba/sealer/image/cache"
	"github.com/alibaba/sealer/image/store"
)

func (d DefaultImageService) BuildImageCache() (Cache, error) {
	ls, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, fmt.Errorf("failed to build image cache, err: %s", err)
	}
	fs, err := store.NewFSStoreBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to init store backend for image cache, err: %s", err)
	}
	imageStore, err := cache.NewImageStore(fs, ls)
	if err != nil {
		return nil, fmt.Errorf("failed to init image store for image cache, err: %s", err)
	}

	return cache.NewLocalImageCache(imageStore)
}
