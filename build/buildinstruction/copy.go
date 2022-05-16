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

package buildinstruction

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/utils/os/fs"

	"github.com/opencontainers/go-digest"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/pkg/image/cache"
	"github.com/sealerio/sealer/pkg/image/store"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/collector"
)

const ArchReg = "${ARCH}"

type CopyInstruction struct {
	src       string
	dest      string
	platform  v1.Platform
	rawLayer  v1.Layer
	fs        store.Backend
	collector collector.Collector
}

func (c CopyInstruction) Exec(execContext ExecContext) (out Out, err error) {
	var (
		hitCache bool
		chainID  cache.ChainID
		cacheID  digest.Digest
		layerID  digest.Digest
		src      = c.src
	)
	defer func() {
		out.ContinueCache = hitCache
		out.ParentID = chainID
	}()

	src = strings.Replace(src, ArchReg, c.platform.Architecture, -1)
	if !isRemoteSource(src) {
		cacheID, err = GenerateSourceFilesDigest(execContext.BuildContext, src)
		if err != nil {
			logger.Warn("failed to generate src digest,discard cache,%s", err)
		}

		if execContext.ContinueCache {
			hitCache, layerID, chainID = tryCache(execContext.ParentID, c.rawLayer, execContext.CacheSvc, execContext.Prober, cacheID)
			// we hit the cache, so we will reuse the layerID layer.
			if hitCache {
				// update chanid as parentid via defer
				out.LayerID = layerID
				return out, nil
			}
		}
	}

	tmp, err := fs.NewFilesystem().MkTmpdir()
	if err != nil {
		return out, fmt.Errorf("failed to create tmp dir %s:%v", tmp, err)
	}

	err = c.collector.Collect(execContext.BuildContext, src, filepath.Join(tmp, c.dest))
	if err != nil {
		return out, fmt.Errorf("failed to collect files to temp dir %s, err: %v", tmp, err)
	}
	// if we come here, its new layer need set cache id .
	layerID, err = execContext.LayerStore.RegisterLayerForBuilder(tmp)
	if err != nil {
		return out, fmt.Errorf("failed to register copy layer, err: %v", err)
	}

	if setErr := c.setCacheID(layerID, cacheID.String()); setErr != nil {
		logger.Warn("failed to set cache for copy layer err: %v", err)
	}

	out.LayerID = layerID
	return out, nil
}

// SetCacheID This function only has meaning for copy layers
func (c CopyInstruction) setCacheID(layerID digest.Digest, cID string) error {
	return c.fs.SetMetadata(layerID, common.CacheID, []byte(cID))
}

func NewCopyInstruction(ctx InstructionContext) (*CopyInstruction, error) {
	f, err := store.NewFSStoreBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to init store backend, err: %s", err)
	}
	src, dest := ParseCopyLayerContent(ctx.CurrentLayer.Value)
	c, err := collector.NewCollector(src)
	if err != nil {
		return nil, fmt.Errorf("failed to init copy Collector, err: %s", err)
	}

	return &CopyInstruction{
		platform:  ctx.Platform,
		fs:        f,
		rawLayer:  *ctx.CurrentLayer,
		src:       src,
		dest:      dest,
		collector: c,
	}, nil
}
