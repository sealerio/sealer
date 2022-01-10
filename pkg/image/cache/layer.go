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
	"strings"

	"github.com/opencontainers/go-digest"
)

type Layer struct {
	// cacheID for layerdb/layersid/cacheID, we will load the content only for COPY layer
	CacheID string `json:"cache_id"`
	// same as v1Layer type
	Type string `json:"type"`
	// same as v1Layer value
	Value string `json:"value"`
}

func (l *Layer) String() string {
	return strings.TrimSpace(l.CacheID) + ":" + strings.TrimSpace(l.Type) + ":" + strings.TrimSpace(l.Value)
}

func (l *Layer) ChainID(parentID ChainID) (ChainID, error) {
	if parentID.String() == "" {
		return ChainID(digest.FromString(l.String())), nil
	}
	return ChainID(digest.FromString(parentID.String() + ":" + l.String())), nil
}
