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
	"os"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/utils/os/fs"

	"github.com/opencontainers/go-digest"
	fsutil "github.com/tonistiigi/fsutil/copy"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/pkg/image/cache"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/archive"
	"github.com/sealerio/sealer/utils/collector"
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
	return true, cacheLayerID, cID
}

func GenerateSourceFilesDigest(root, src string) (digest.Digest, error) {
	m, err := fsutil.ResolveWildcards(root, src, true)
	if err != nil {
		return "", err
	}

	// wrong wildcards: no such file or directory
	if len(m) == 0 {
		return "", fmt.Errorf("%s not found", src)
	}

	if len(m) == 1 {
		return generateDigest(filepath.Join(root, src))
	}

	tmp, err := fs.NewFilesystem().MkTmpdir()
	if err != nil {
		return "", fmt.Errorf("failed to create tmp dir %s:%v", tmp, err)
	}

	defer func() {
		if err = os.RemoveAll(tmp); err != nil {
			logger.Warn(err)
		}
	}()

	xattrErrorHandler := func(dst, src, key string, err error) error {
		logger.Warn(err)
		return nil
	}
	opt := []fsutil.Opt{
		fsutil.WithXAttrErrorHandler(xattrErrorHandler),
	}

	for _, s := range m {
		if err := fsutil.Copy(context.TODO(), root, s, tmp, filepath.Base(s), opt...); err != nil {
			return "", err
		}
	}

	return generateDigest(tmp)
}

func generateDigest(path string) (digest.Digest, error) {
	layerDgst, _, err := archive.TarCanonicalDigest(path)
	if err != nil {
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

func ParseCopyLayerContent(layerValue string) (src, dst string) {
	dst = strings.Fields(layerValue)[1]
	for _, p := range []string{"./", "/"} {
		dst = strings.TrimPrefix(dst, p)
	}
	dst = strings.TrimSuffix(dst, "/")
	src = strings.Fields(layerValue)[0]
	return
}

func isRemoteSource(src string) bool {
	if collector.IsURL(src) || collector.IsGitURL(src) {
		return true
	}
	return false
}
