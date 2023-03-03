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
	"os"
	"path/filepath"
	"strings"

	reference2 "github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/sealerio/sealer/build/layerutils/charts"
	manifest "github.com/sealerio/sealer/build/layerutils/manifests"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/define/application"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/image/save"
	"github.com/sealerio/sealer/pkg/rootfs"
	v1 "github.com/sealerio/sealer/types/api/v1"
	osi "github.com/sealerio/sealer/utils/os"
)

// TODO: update the variable name
var (
	copyToManifests   = "manifests"
	copyToChart       = "charts"
	copyToImageList   = "imageList"
	copyToApplication = "application"
)

type parseContainerImageStringSliceFunc func(srcPath string) ([]string, error)
type parseContainerImageListFunc func(srcPath string) ([]*v12.ContainerImage, error)

var parseContainerImageStringSliceFuncMap = map[string]func(srcPath string) ([]string, error){
	copyToManifests:   parseYamlImages,
	copyToChart:       parseChartImages,
	copyToImageList:   parseRawImageList,
	copyToApplication: WrapParseContainerImageList2StringSlice(parseApplicationImages),
}

var parseContainerImageListFuncMap = map[string]func(srcPath string) ([]*v12.ContainerImage, error){
	copyToManifests:   WrapParseStringSlice2ContainerImageList(parseYamlImages),
	copyToChart:       WrapParseStringSlice2ContainerImageList(parseChartImages),
	copyToImageList:   WrapParseStringSlice2ContainerImageList(parseRawImageList),
	copyToApplication: parseApplicationImages,
}

type Registry struct {
	platform v1.Platform
	puller   save.ImageSave
}

func NewRegistry(platform v1.Platform) *Registry {
	ctx := context.Background()
	return &Registry{
		platform: platform,
		puller:   save.NewImageSaver(ctx),
	}
}

func (r *Registry) SaveImages(rootfs string, containerImages []string) error {
	return r.puller.SaveImages(containerImages, filepath.Join(rootfs, common.RegistryDirName), r.platform)
}

func WrapParseStringSlice2ContainerImageList(parseFunc parseContainerImageStringSliceFunc) func(srcPath string) ([]*v12.ContainerImage, error) {
	return func(srcPath string) ([]*v12.ContainerImage, error) {
		images, err := parseFunc(srcPath)
		if err != nil {
			return nil, err
		}
		var containerImageList []*v12.ContainerImage
		for _, image := range images {
			containerImageList = append(containerImageList, &v12.ContainerImage{
				Image:   image,
				AppName: "",
			})
		}
		return containerImageList, nil
	}
}

func WrapParseContainerImageList2StringSlice(parseFunc parseContainerImageListFunc) func(srcPath string) ([]string, error) {
	return func(srcPath string) ([]string, error) {
		containerImageList, err := parseFunc(srcPath)
		if err != nil {
			return nil, err
		}
		return v12.GetImageSliceFromContainerImageList(containerImageList), nil
	}
}

func ParseContainerImageList(srcPath string) ([]*v12.ContainerImage, error) {
	eg, _ := errgroup.WithContext(context.Background())

	var containerImageList []*v12.ContainerImage
	for t, p := range parseContainerImageListFuncMap {
		dispatchType := t
		parse := p
		eg.Go(func() error {
			parsedImageList, err := parse(srcPath)
			if err != nil {
				return fmt.Errorf("failed to parse images from %s: %v", dispatchType, err)
			}
			for _, image := range parsedImageList {
				img, err := reference2.ParseNormalizedNamed(image.Image)
				if err != nil {
					continue
				}
				containerImageList = append(containerImageList, &v12.ContainerImage{
					Image:    img.String(),
					AppName:  image.AppName,
					Platform: image.Platform,
				})
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return containerImageList, nil
}

func ParseContainerImageSlice(srcPath string) ([]string, error) {
	eg, _ := errgroup.WithContext(context.Background())

	var images []string
	for t, p := range parseContainerImageStringSliceFuncMap {
		dispatchType := t
		parse := p
		eg.Go(func() error {
			ima, err := parse(srcPath)
			if err != nil {
				return fmt.Errorf("failed to parse images from %s: %v", dispatchType, err)
			}
			images = append(images, ima...)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return images, nil
}

func parseApplicationImages(srcPath string) ([]*v12.ContainerImage, error) {
	applicationPath := filepath.Clean(filepath.Join(srcPath, rootfs.GlobalManager.App().Root()))

	if !osi.IsFileExist(applicationPath) {
		return nil, nil
	}

	var (
		containerImageList []*v12.ContainerImage
		err                error
	)

	entries, err := os.ReadDir(applicationPath)
	if err != nil {
		return nil, errors.Wrap(err, "error in readdir in parseApplicationImages")
	}
	for _, entry := range entries {
		name := entry.Name()
		appPath := filepath.Join(applicationPath, name)
		if entry.IsDir() {
			if !isChartArtifactEnough(appPath) {
				imagesTmp, err := parseApplicationKubeImages(appPath)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse container image list for app(%s) with type(%s)",
						name, application.KubeApp)
				}
				for _, image := range imagesTmp {
					containerImageList = append(containerImageList, &v12.ContainerImage{
						Image:   image,
						AppName: name,
					})
				}
				continue
			}

			imagesTmp, err := parseApplicationHelmImages(appPath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse container image list for app(%s) with type(%s)",
					name, application.HelmApp)
			}
			for _, image := range imagesTmp {
				containerImageList = append(containerImageList, &v12.ContainerImage{
					Image:   image,
					AppName: name,
				})
			}
		}
	}

	return containerImageList, nil
}

func parseApplicationHelmImages(helmPath string) ([]string, error) {
	if !osi.IsFileExist(helmPath) {
		return nil, nil
	}

	var images []string

	imageSearcher, err := charts.NewCharts()
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(helmPath, func(path string, f fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !f.IsDir() {
			return nil
		}

		if isChartArtifactEnough(path) {
			imgs, err := imageSearcher.ListImages(path)
			if err != nil {
				return err
			}

			images = append(images, imgs...)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return FormatImages(images), nil
}

func parseApplicationKubeImages(kubePath string) ([]string, error) {
	if !osi.IsFileExist(kubePath) {
		return nil, nil
	}
	var images []string
	imageSearcher, err := manifest.NewManifests()
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(kubePath, func(path string, f fs.FileInfo, err error) error {
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

func parseChartImages(srcPath string) ([]string, error) {
	chartsPath := filepath.Join(srcPath, copyToChart)
	if !osi.IsFileExist(chartsPath) {
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

		if isChartArtifactEnough(path) {
			ima, err := imageSearcher.ListImages(path)
			if err != nil {
				return err
			}
			images = append(images, ima...)
			return filepath.SkipDir
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
	if !osi.IsFileExist(manifestsPath) {
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
	if !osi.IsFileExist(imageListFilePath) {
		return nil, nil
	}

	images, err := osi.NewFileReader(imageListFilePath).ReadLines()
	if err != nil {
		return nil, fmt.Errorf("failed to read file content %s:%v", imageListFilePath, err)
	}
	return FormatImages(images), nil
}

var isChartArtifactEnough = func(path string) bool {
	return osi.IsFileExist(filepath.Join(path, "Chart.yaml")) &&
		osi.IsFileExist(filepath.Join(path, "values.yaml")) &&
		osi.IsFileExist(filepath.Join(path, "templates"))
}
