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
	"github.com/alibaba/sealer/build/buildkit"
	"github.com/alibaba/sealer/build/buildkit/buildimage"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image/reference"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type liteBuilder struct {
	buildType    string
	noCache      bool
	noBase       bool
	imageNamed   reference.Named
	context      string
	kubeFileName string
	buildArgs    map[string]string
	baseLayers   []v1.Layer
	rawImage     *v1.Image
	executor     buildimage.Executor
	saver        buildimage.ImageSaver
}

func (l liteBuilder) Build(name string, context string, kubefileName string) error {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return err
	}
	l.imageNamed = named

	absContext, absKubeFile, err := buildkit.ParseBuildArgs(context, kubefileName)
	if err != nil {
		return err
	}
	l.kubeFileName = absKubeFile

	err = buildkit.ValidateContextDirectory(absContext)
	if err != nil {
		return err
	}
	l.context = absContext

	rawImage, baseLayers, err := buildimage.NewBuildImageByKubefile(absKubeFile)
	if err != nil {
		return err
	}
	l.rawImage, l.baseLayers = rawImage, baseLayers

	executor, err := buildimage.NewLayerExecutor(baseLayers, l.buildType)
	if err != nil {
		return err
	}
	l.executor = executor

	saver, err := buildimage.NewImageSaver(l.buildType)
	if err != nil {
		return err
	}
	l.saver = saver

	pipLine, err := l.GetBuildPipeLine()
	if err != nil {
		return err
	}

	for _, f := range pipLine {
		if err = f(); err != nil {
			return err
		}
	}
	return nil
}

func (l liteBuilder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	buildPipeline = append(buildPipeline,
		l.ExecBuild,
		l.SaveBuildImage,
		l.Cleanup,
	)
	return buildPipeline, nil
}

func (l liteBuilder) ExecBuild() error {
	// merge args with build context
	for k, v := range l.buildArgs {
		l.rawImage.Spec.ImageConfig.Args.Current[k] = v
	}

	ctx := buildimage.Context{
		BuildContext: l.context,
		UseCache:     !l.noCache,
		BuildArgs:    l.rawImage.Spec.ImageConfig.Args.Current,
	}

	layers, err := l.executor.Execute(ctx, l.rawImage.Spec.Layers[1:])
	if err != nil {
		return err
	}

	l.rawImage.Spec.Layers = layers
	return nil
}

func (l liteBuilder) SaveBuildImage() error {
	l.rawImage.Name = l.imageNamed.Raw()
	if l.noBase {
		l.rawImage.Spec.ImageConfig.ImageType = common.AppImage
		l.rawImage.Spec.ImageConfig.Cmd.Parent = nil
		l.rawImage.Spec.ImageConfig.Args.Parent = nil
		l.rawImage.Spec.Layers = l.rawImage.Spec.Layers[len(l.baseLayers):]
	}

	err := l.saver.Save(l.rawImage)
	if err != nil {
		return err
	}
	logger.Info("save image %s to image system success", l.rawImage.Name)
	return nil
}

func (l liteBuilder) Cleanup() error {
	return l.executor.Cleanup()
}

func NewLiteBuilder(config *Config) (Interface, error) {
	return &liteBuilder{
		buildType: config.BuildType,
		noCache:   config.NoCache,
		noBase:    config.NoBase,
		buildArgs: config.BuildArgs,
	}, nil
}
