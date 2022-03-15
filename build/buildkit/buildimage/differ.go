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
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/build/buildkit/buildinstruction"
	"github.com/alibaba/sealer/build/buildkit/layerutils/charts"
	manifest "github.com/alibaba/sealer/build/buildkit/layerutils/manifests"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/image/save"
	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/utils"
	"golang.org/x/sync/errgroup"
)

var (
	copyToManifests = "manifests"
	copyToChart     = "charts"
	copyToImageList = "imageList"
	dispatch        map[string]func(srcPath string) ([]string, error)
)

func init() {
	dispatch = map[string]func(srcPath string) ([]string, error){
		copyToManifests: parseYamlImages,
		copyToChart:     parseChartImages,
		copyToImageList: parseRawImageList,
	}
}

type registry struct {
	puller save.ImageSave
}

func (r registry) Process(src, dst buildinstruction.MountTarget) error {
	srcPath := src.GetMountTarget()
	rootfs := dst.GetMountTarget()
	eg, _ := errgroup.WithContext(context.Background())

	var images []string
	for t, p := range dispatch {
		dispatchType := t
		parse := p
		eg.Go(func() error {
			ima, err := parse(srcPath)
			if err != nil {
				return fmt.Errorf("failed to parse images from %s error is : %v", dispatchType, err)
			}
			images = append(images, ima...)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}
	plat := runtime.GetCloudImagePlatform(rootfs)
	return r.puller.SaveImages(images, filepath.Join(rootfs, common.RegistryDirName), plat)
}

func NewRegistryDiffer() Differ {
	ctx := context.Background()
	return registry{
		puller: save.NewImageSaver(ctx),
	}
}

func parseChartImages(srcPath string) ([]string, error) {
	chartsPath := filepath.Join(srcPath, copyToChart)
	if !utils.IsExist(chartsPath) {
		return nil, nil
	}

	var images []string
	imageSearcher, err := charts.NewCharts()
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(chartsPath, func(path string, f fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !f.IsDir() {
			return nil
		}

		if utils.IsExist(filepath.Join(path, "Chart.yaml")) && utils.IsExist(filepath.Join(path, "values.yaml")) &&
			utils.IsExist(filepath.Join(path, "templates")) {
			ima, err := imageSearcher.ListImages(path)
			if err != nil {
				return err
			}
			images = append(images, ima...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return FormatImages(images), nil
}

func parseYamlImages(srcPath string) ([]string, error) {
	manifestsPath := filepath.Join(srcPath, copyToManifests)
	if !utils.IsExist(manifestsPath) {
		return nil, nil
	}
	var images []string

	imageSearcher, err := manifest.NewManifests()
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(manifestsPath, func(path string, f fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext != ".yaml" && ext != ".yml" && ext != ".tmpl" {
			return nil
		}

		ima, err := imageSearcher.ListImages(path)

		if err != nil {
			return err
		}
		images = append(images, ima...)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return FormatImages(images), nil
}

func parseRawImageList(srcPath string) ([]string, error) {
	imageListFilePath := filepath.Join(srcPath, copyToManifests, copyToImageList)
	if !utils.IsExist(imageListFilePath) {
		return nil, nil
	}

	images, err := utils.ReadLines(imageListFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content %s:%v", imageListFilePath, err)
	}
	return FormatImages(images), nil
}

type metadata struct {
}

func (m metadata) Process(src, dst buildinstruction.MountTarget) error {
	// check "KubeVersion" of Chart.yaml under charts dir,to overwrite the metadata.
	srcPath := src.GetMountTarget()
	rootfs := dst.GetMountTarget()
	kv := getKubeVersion(srcPath)
	if kv == "" {
		return nil
	}

	md, err := m.loadMetadata(srcPath, rootfs)
	if err != nil {
		return err
	}

	if md.KubeVersion == kv {
		return nil
	}
	md.KubeVersion = kv
	mf := filepath.Join(rootfs, common.DefaultMetadataName)
	if err = utils.MarshalJSONToFile(mf, md); err != nil {
		return fmt.Errorf("failed to set image Metadata file, err: %v", err)
	}

	return nil
}

func (m metadata) loadMetadata(srcPath, rootfs string) (*runtime.Metadata, error) {
	// if Metadata file existed in srcPath, load and marshal to check the legality of it's content.
	// if not, use rootfs Metadata.
	smd, err := runtime.LoadMetadata(srcPath)
	if err != nil {
		return nil, err
	}
	if smd != nil {
		return smd, nil
	}

	md, err := runtime.LoadMetadata(rootfs)
	if err != nil {
		return nil, err
	}

	if md != nil {
		return md, nil
	}
	return nil, fmt.Errorf("failed to load rootfs Metadata")
}

func NewMetadataDiffer() Differ {
	return metadata{}
}
