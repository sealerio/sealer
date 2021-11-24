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

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/store"

	"github.com/alibaba/sealer/build/buildkit/buildlayer"
	"github.com/alibaba/sealer/image/cache"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/opencontainers/go-digest"
)

type CopyInstruction struct {
	src          string
	dest         string
	rawLayer     v1.Layer
	layerHandler buildlayer.LayerHandler
	fs           store.Backend
	mounter      MountTarget
}

func (c CopyInstruction) Exec(execContext ExecContext) (out Out, err error) {
	// pre handle layer content
	if c.layerHandler != nil {
		err = c.layerHandler.LayerValueHandler(execContext.BuildContext, execContext.SealerDocker)
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

	// specially for copy command, we would generate digest of src file as cacheID.
	// for every copy, we will hard link (make digest consistent) the copy source files, and generate a digest for those files
	// and use the cacheID try if it can hit the cache

	cacheID, err = GenerateSourceFilesDigest(filepath.Join(execContext.BuildContext, c.src))
	if err != nil {
		logger.Warn("failed to generate src digest, discard cache, err: %s", err)
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

	err = c.mounter.TempMount()
	if err != nil {
		return out, err
	}
	defer c.mounter.CleanUp()
	err = utils.SetRootfsBinToSystemEnv(c.mounter.GetMountTarget())
	if err != nil {
		return out, fmt.Errorf("failed to set temp rootfs %s to system $PATH : %v", c.mounter.GetMountTarget(), err)
	}

	err = c.copyFiles(execContext.BuildContext, c.src, c.dest, c.mounter.GetMountTarget())
	if err != nil {
		return out, fmt.Errorf("failed to copy files to temp dir %s, err: %v", c.mounter.GetMountTarget(), err)
	}
	// if we come here, its new layer need set cacheid .
	layerID, err = execContext.LayerStore.RegisterLayerForBuilder(c.mounter.GetMountUpper())
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
	absSrc := filepath.Join(buildContext, rawSrcFileName)
	if !utils.IsExist(absSrc) {
		return fmt.Errorf("failed to stat file %s at copy stage", absSrc)
	}

	return utils.RecursionCopy(absSrc, filepath.Join(paresCopyDestPath(rawDstFileName, tempBuildDir), filepath.Base(rawSrcFileName)))
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
	lowerLayers := GetBaseLayersPath(ctx.BaseLayers)
	target, err := NewMountTarget("", "", lowerLayers)
	if err != nil {
		return nil, err
	}
	layerValue := strings.Fields(ctx.CurrentLayer.Value)
	return &CopyInstruction{
		fs:           fs,
		mounter:      *target,
		layerHandler: buildlayer.ParseLayerContent(ctx.CurrentLayer),
		rawLayer:     *ctx.CurrentLayer,
		src:          layerValue[0],
		dest:         layerValue[1],
	}, nil
}
