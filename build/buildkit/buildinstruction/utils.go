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

	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/image/cache"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/archive"
	"github.com/opencontainers/go-digest"
)

func tryCache(parentID cache.ChainID,
	layer v1.Layer,
	cacheService cache.Service,
	prober image.Prober,
	srcFilesDgst digest.Digest) (hitCache bool, layerID digest.Digest, chainID cache.ChainID) {
	var err error
	cacheLayer := cacheService.NewCacheLayer(layer, srcFilesDgst)
	cacheLayerID, err := prober.Probe(parentID.String(), &cacheLayer)
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
	fmt.Println("chain id ", cID)
	return true, cacheLayerID, cID
}

func paresCopyDestPath(rawDstFileName, tempBuildDir string) string {
	// pares copy dest,default workdir is rootfs
	//copy . . = $rootfs
	// copy abc .= $rootfs/abc
	// copy abc ./manifest = $rootfs/manifest/abc
	// copy abc charts = $rootfs/charts/abc
	// copy abc charts/test = $rootfs/charts/test/abc
	// copy abc /tmp = $rootfs/tmp/abc
	dst := rawDstFileName
	if dst == "." || dst == "./" || dst == "/" || dst == "/." {
		return tempBuildDir
	}

	for _, p := range []string{"./", "/"} {
		dst = strings.TrimPrefix(dst, p)
	}
	return filepath.Join(tempBuildDir, dst)
}

func GenerateSourceFilesDigest(path string) (digest.Digest, error) {
	layerDgst, _, err := archive.TarCanonicalDigest(path)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	return layerDgst, nil
}

// GetBaseLayersPath used in build stage, where the image still has from layer
func GetBaseLayersPath(layers []v1.Layer) (res []string) {
	for _, layer := range layers {
		if layer.ID != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.ID.Hex()))
		}
	}
	return res
}
