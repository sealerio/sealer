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

package lite

import (
	"github.com/alibaba/sealer/build/buildkit"
	"github.com/alibaba/sealer/build/buildkit/buildimage"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image/reference"
)

type Builder struct {
	BuildType    string
	NoCache      bool
	NoBase       bool
	ImageNamed   reference.Named
	Context      string
	KubeFileName string
	BuildImage   buildimage.Interface
}

func (l *Builder) Build(name string, context string, kubefileName string) error {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return err
	}
	l.ImageNamed = named

	absContext, absKubeFile, err := buildkit.ParseBuildArgs(context, kubefileName)
	if err != nil {
		return err
	}
	l.KubeFileName = absKubeFile

	err = buildkit.ValidateContextDirectory(absContext)
	if err != nil {
		return err
	}
	l.Context = absContext

	bi, err := buildimage.NewBuildImage(absKubeFile, l.BuildType)
	if err != nil {
		return err
	}
	l.BuildImage = bi

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

func (l *Builder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	buildPipeline = append(buildPipeline,
		l.PreCheck,
		l.ExecBuild,
		l.SaveBuildImage,
		l.Cleanup,
	)
	return buildPipeline, nil
}

func (l *Builder) PreCheck() error {
	return nil
}

func (l *Builder) ExecBuild() error {
	ctx := buildimage.Context{
		BuildContext: l.Context,
		UseCache:     !l.NoCache,
	}

	return l.BuildImage.ExecBuild(ctx)
}

func (l *Builder) SaveBuildImage() error {
	imageName := l.ImageNamed.Raw()

	err := l.BuildImage.SaveBuildImage(imageName, buildimage.SaveOpts{
		WithoutBase: l.NoBase,
	})
	if err != nil {
		return err
	}
	logger.Info("save image %s to image system success !", imageName)
	return nil
}

func (l *Builder) Cleanup() error {
	return l.BuildImage.Cleanup()
}
