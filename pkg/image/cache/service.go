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

package cache

import (
	"fmt"

	"github.com/opencontainers/go-digest"

	v1 "github.com/sealerio/sealer/types/api/v1"
)

type Service interface {
	NewCacheLayer(layer v1.Layer, cacheID digest.Digest) Layer

	CalculateChainID(layers interface{}) (ChainID, error)
}

type service struct {
}

func (s *service) NewCacheLayer(layer v1.Layer, cacheID digest.Digest) Layer {
	return Layer{
		CacheID: cacheID.String(),
		Type:    layer.Type,
		Value:   layer.Value,
	}
}

func (s *service) CalculateChainID(layers interface{}) (ChainID, error) {
	switch ls := layers.(type) {
	case []Layer:
		return CalculateCacheID(ls)
	default:
		return "", fmt.Errorf("do not support calculate chain ID on %v", ls)
	}
}

func NewService() (Service, error) {
	return &service{}, nil
}
