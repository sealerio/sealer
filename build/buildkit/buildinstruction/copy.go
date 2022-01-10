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
	"context"
	"fmt"
	"path/filepath"

	fsutil "github.com/tonistiigi/fsutil/copy"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/image/store"

	"github.com/opencontainers/go-digest"

	"github.com/alibaba/sealer/build/buildkit/buildlayer"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image/cache"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type CopyInstruction struct {
	src          string
	dest         string
	rawLayer     v1.Layer
	layerHandler buildlayer.LayerHandler
	fs           store.Backend
}

func (c CopyInstruction) Exec(execContext ExecContext) (out Out, err error) {
	// pre handle layer content
	if c.layerHandler != nil {
		err = c.layerHandler.LayerValueHandler(execContext.BuildContext, c.rawLayer)
		if err != nil {
			return out, err
		}
	}

	var (
		hitCache bool
		chainID  cache.ChainID
		cacheID  digest.Digest
		layerID  digest.Digest
	)
	defer func() {
		out.ContinueCache = hitCache
		out.ParentID = chainID
	}()

	cacheID, err = GenerateSourceFilesDigest(execContext.BuildContext, c.src)
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

	tmp, err := utils.MkTmpdir()
	if err != nil {
		return out, fmt.Errorf("failed to create tmp dir %s:%v", tmp, err)
	}

	err = c.copyFiles(execContext.BuildContext, c.src, c.dest, tmp)
	if err != nil {
		return out, fmt.Errorf("failed to copy files to temp dir %s, err: %v", tmp, err)
	}
	// if we come here, its new layer need set cacheid .
	layerID, err = execContext.LayerStore.RegisterLayerForBuilder(tmp)
	if err != nil {
		return out, fmt.Errorf("failed to register copy layer, err: %v", err)
	}

	if setErr := c.SetCacheID(layerID, cacheID.String()); setErr != nil {
		logger.Warn("set cache failed layer: %v, err: %v", c.rawLayer, err)
	}

	out.LayerID = layerID
	return out, nil
}

func (c CopyInstruction) copyFiles(buildContext, rawSrcFileName, rawDstFileName, tempBuildDir string) error {
	xattrErrorHandler := func(dst, src, key string, err error) error {
		logger.Warn(err)
		return nil
	}
	opt := []fsutil.Opt{
		fsutil.WithXAttrErrorHandler(xattrErrorHandler),
	}

	dstRoot := paresCopyDestPath(rawDstFileName, tempBuildDir)

	m, err := fsutil.ResolveWildcards(buildContext, rawSrcFileName, true)
	if err != nil {
		return err
	}

	if len(m) == 0 {
		return fmt.Errorf("%s not found", rawSrcFileName)
	}
	for _, s := range m {
		if err := fsutil.Copy(context.TODO(), buildContext, s, dstRoot, filepath.Base(s), opt...); err != nil {
			return err
		}
	}
	return nil
}

// SetCacheID This function only has meaning for copy layers
func (c CopyInstruction) SetCacheID(layerID digest.Digest, cID string) error {
	return c.fs.SetMetadata(layerID, common.CacheID, []byte(cID))
}

func NewCopyInstruction(ctx InstructionContext) (*CopyInstruction, error) {
	fs, err := store.NewFSStoreBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to init store backend, err: %s", err)
	}
	src, dest := buildlayer.ParseCopyLayerContent(ctx.CurrentLayer.Value)
	return &CopyInstruction{
		fs:           fs,
		layerHandler: buildlayer.ParseLayerContent(ctx.Rootfs, ctx.CurrentLayer),
		rawLayer:     *ctx.CurrentLayer,
		src:          src,
		dest:         dest,
	}, nil
}
