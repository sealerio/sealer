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

package build

import (
	"fmt"
	"github.com/alibaba/sealer/pkg/command"
	"github.com/alibaba/sealer/pkg/image"
	cache2 "github.com/alibaba/sealer/pkg/image/cache"
	"github.com/alibaba/sealer/pkg/image/store"
	"github.com/alibaba/sealer/pkg/logger"
	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils/archive"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/opencontainers/go-digest"
)

type handlerContext struct {
	buildContext  string
	continueCache bool
	cacheSvc      cache2.Service
	prober        image.Prober
	parentID      cache2.ChainID
	ignoreError   bool
}

type handler struct {
	hc         handlerContext
	layerStore store.LayerStore
}

func (h *handler) handleCopyCmd(layer v1.Layer) (layerID digest.Digest, cacheID digest.Digest, err error) {
	//tempHardLinkDir, err := utils.MkTmpdir()
	//if err != nil {
	//	return "", "", fmt.Errorf("failed to create temp hard link dir %s, err: %v", tempHardLinkDir, err)
	//}
	var (
		hitCache bool
		chainID  cache2.ChainID
	)
	defer func() {
		h.hc.continueCache = hitCache
		h.hc.parentID = chainID
		//utils.CleanDir(tempHardLinkDir)
	}()
	//err = h.hardLinkFiles(strings.Fields(layer.Value)[0], strings.Fields(layer.Value)[1], tempHardLinkDir)
	//if err != nil {
	//	return "", "", fmt.Errorf("failed to hard link files, err: %v", err)
	//}

	// specially for copy command, we would generate digest of src file as cacheID.
	// for every copy, we will hard link (make digest consistent) the copy source files, and generate a digest for those files
	// and use the cacheID try if it can hit the cache
	// TODO is there any way to generate source digest not on src files directly?
	// hard link can't do this, because it will occurs cross device issue
	cacheID, err = h.generateSourceFilesDigest(filepath.Join(h.hc.buildContext, strings.Fields(layer.Value)[0]))
	//cacheID, err = h.generateSourceFilesDigest(tempHardLinkDir)
	if err != nil {
		logger.Warn("failed to generate src digest, discard cache, err: %s", err)
	}

	if h.hc.continueCache {
		hitCache, layerID, chainID = h.tryCache(h.hc.parentID, layer, h.hc.cacheSvc, cacheID)
		// we hit the cache, so we will reuse the layerID layer.
		if hitCache {
			// update chanid as parentid
			return layerID, "", nil
		}
	}

	tempCopyDir, err := utils.MkTmpdir()
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp copy dir %s, err: %v", tempCopyDir, err)
	}
	defer utils.CleanDir(tempCopyDir)

	err = h.copyFiles(strings.Fields(layer.Value)[0], strings.Fields(layer.Value)[1], tempCopyDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to copy files to temp dir %s, err: %v", tempCopyDir, err)
	}
	// if we come here, cache is no longer possible
	layerID, err = h.layerStore.RegisterLayerForBuilder(tempCopyDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to register copy layer, err: %v", err)
	}

	return layerID, cacheID, nil
}

func (h *handler) handleCMDRUNCmd(layer v1.Layer, lowerLayers ...string) (layerID digest.Digest, err error) {
	var (
		hitCache bool
		chanID   cache2.ChainID
	)
	defer func() {
		h.hc.continueCache = hitCache
		h.hc.parentID = chanID
	}()

	if h.hc.continueCache {
		hitCache, layerID, chanID = h.tryCache(h.hc.parentID, layer, h.hc.cacheSvc, "")
		if hitCache {
			// update chanid as parentid
			return layerID, nil
		}
	}

	target, err := NewMountTarget("", "", lowerLayers)
	if err != nil {
		return "", err
	}
	defer target.CleanUp()

	err = target.TempMount()
	if err != nil {
		return "", err
	}

	cmd := fmt.Sprintf(common.CdAndExecCmd, target.GetMountTarget(), layer.Value)
	output, err := command.NewSimpleCommand(cmd).Exec()
	logger.Info(output)

	if err != nil {
		if h.hc.ignoreError {
			logger.Warn(fmt.Sprintf("failed to exec %s, err: %v", cmd, err))
			return "", nil
		}
		return "", fmt.Errorf("failed to exec %s, err: %v", cmd, err)
	}

	// cmd do not contains layer ,so no need to calculate layer
	if layer.Type != common.CMDCOMMAND {
		return h.layerStore.RegisterLayerForBuilder(target.GetMountUpper())
	}

	return "", nil
}

//func (h *handler) hardLinkFiles(srcFileName, dstFileName, tempBuildDir string) error {
//	var (
//		src = filepath.Join(h.hc.buildContext, srcFileName)
//		dst string
//	)
//
//	fi, err := os.Stat(src)
//	if err != nil {
//		return fmt.Errorf("failed to stat file %s at hard link, err: %s", src, err)
//	}
//
//	if fi.IsDir() {
//		dst = filepath.Join(tempBuildDir, dstFileName, filepath.Base(src))
//	} else {
//		dst = filepath.Join(tempBuildDir, dstFileName, srcFileName)
//	}
//	return utils.RecursionHardLink(src, dst)
//}

func (h *handler) copyFiles(srcFileName, dstFileName, tempBuildDir string) error {
	var (
		src = filepath.Join(h.hc.buildContext, srcFileName)
		dst string
	)

	fi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat file %s at copy files, err: %s", src, err)
	}

	if fi.IsDir() {
		//default workdir is rootfs,so if copy dst is ".", name it as filepath.Base(srcFileName)
		if dstFileName == "." {
			dstFileName = filepath.Base(srcFileName)
		}
		dst = filepath.Join(tempBuildDir, dstFileName)
	} else {
		dst = filepath.Join(tempBuildDir, dstFileName, srcFileName)
	}
	return utils.RecursionCopy(src, dst)
}

func (h *handler) generateSourceFilesDigest(path string) (digest.Digest, error) {
	layerDgst, _, err := archive.TarCanonicalDigest(path)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	return layerDgst, nil
}

func (h *handler) tryCache(parentID cache2.ChainID, layer v1.Layer, cacheService cache2.Service, srcFilesDgst digest.Digest) (hitCache bool, layerID digest.Digest, chainID cache2.ChainID) {
	var err error
	cacheLayer := cacheService.NewCacheLayer(layer, srcFilesDgst)
	cacheLayerID, err := h.hc.prober.Probe(parentID.String(), &cacheLayer)
	if err != nil {
		logger.Debug("failed to probe cache for %+v, err: %s", layer, err)
		return false, "", ""
	}
	// cache hit
	logger.Info("---> Using cache %v", cacheLayerID)
	//layer.ID = cacheLayerID
	cID, err := cacheLayer.ChainID(parentID)
	if err != nil {
		return false, "", ""
	}
	return true, cacheLayerID, cID
}
