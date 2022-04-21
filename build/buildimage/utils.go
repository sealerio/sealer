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

package buildimage

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/build/buildinstruction"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/alibaba/sealer/pkg/parser"
	"sigs.k8s.io/yaml"
)

// initImageSpec init default Image metadata
func initImageSpec(kubefile string) (*v1.Image, error) {
	kubeFile, err := utils.ReadAll(kubefile)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubefile: %v", err)
	}

	rawImage, err := parser.NewParse().Parse(kubeFile)
	if err != nil {
		return nil, err
	}

	layer0 := rawImage.Spec.Layers[0]
	if layer0.Type != common.FROMCOMMAND {
		return nil, fmt.Errorf("first line of kubefile must start with %s", common.FROMCOMMAND)
	}

	return rawImage, nil
}

func loadClusterFile(path string) (*v2.Cluster, error) {
	var cluster v2.Cluster
	rawClusterFile, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	if len(rawClusterFile) == 0 {
		return nil, fmt.Errorf("ClusterFile content is empty")
	}

	if err = yaml.Unmarshal(rawClusterFile, &cluster); err != nil {
		return nil, err
	}

	return &cluster, nil
}

func setClusterFileToImage(cluster *v2.Cluster, image *v1.Image) error {
	clusterData, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	if image.Annotations == nil {
		image.Annotations = make(map[string]string)
	}
	image.Annotations[common.ImageAnnotationForClusterfile] = string(clusterData)
	return nil
}

func getKubeVersion(rootfs string) string {
	chartsPath := filepath.Join(rootfs, "charts")
	if !utils.IsExist(chartsPath) {
		return ""
	}
	return readCharts(chartsPath)
}

func readCharts(chartsPath string) string {
	var kv string
	err := filepath.Walk(chartsPath, func(path string, f fs.FileInfo, err error) error {
		if kv != "" {
			return nil
		}
		if f.IsDir() || f.Name() != "Chart.yaml" {
			return nil
		}
		meta, walkErr := chartutil.LoadChartfile(path)
		if walkErr != nil {
			return walkErr
		}
		if meta.KubeVersion != "" {
			kv = meta.KubeVersion
		}
		return nil
	})

	if err != nil {
		return ""
	}
	return kv
}

func FormatImages(images []string) (res []string) {
	for _, ima := range utils.RemoveDuplicate(images) {
		if ima == "" {
			continue
		}
		if strings.HasPrefix(ima, "#") {
			continue
		}
		res = append(res, trimQuotes(strings.TrimSpace(ima)))
	}
	return
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if c := s[len(s)-1]; s[0] == c && (c == '"' || c == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// GetLayerMountInfo to get rootfs mount info.if not mounted will mount it via base layers.
//1, already mount: runtime docker registry mount info,just get related mount info.
//2, already mount: if exec build cmd failed and return ,need to collect related old mount info
//3, new mount: just mount and return related info.
func GetLayerMountInfo(baseLayers []v1.Layer) (mount.Service, error) {
	var filterArgs = "tmp"
	mountInfos := mount.GetBuildMountInfo(filterArgs)
	lowerLayers := buildinstruction.GetBaseLayersPath(baseLayers)
	for _, info := range mountInfos {
		// if info.Lowers equal lowerLayers,means image already mounted.
		if strings.Join(lowerLayers, ":") == strings.Join(info.Lowers, ":") {
			logger.Info("get mount dir :%s success ", info.Target)
			//nolint
			return mount.NewMountService(info.Target, info.Upper, info.Lowers)
		}
	}

	return mountRootfs(lowerLayers)
}

func mountRootfs(res []string) (mount.Service, error) {
	mounter, err := mount.NewMountService("", "", res)
	if err != nil {
		return nil, err
	}

	err = mounter.TempMount()
	if err != nil {
		return nil, err
	}
	return mounter, nil
}
