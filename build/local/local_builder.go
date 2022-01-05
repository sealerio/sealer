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

package local

import (
	"fmt"
	"time"

	"github.com/alibaba/sealer/build/buildkit"
	"github.com/alibaba/sealer/build/buildkit/buildimage"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/client/k8s"
	"github.com/alibaba/sealer/pkg/image/reference"
	"github.com/alibaba/sealer/utils"
)

// Builder : local builder using local provider to build a cluster image
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
	err := l.InitBuilder(name, context, kubefileName)
	if err != nil {
		return err
	}

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

// InitBuilder pares cmd line parameters and pares Kubefile
func (l *Builder) InitBuilder(name string, context string, kubefileName string) error {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return err
	}

	absContext, absKubeFile, err := buildkit.ParseBuildArgs(context, kubefileName)
	if err != nil {
		return err
	}

	err = buildkit.ValidateContextDirectory(absContext)
	if err != nil {
		return err
	}

	bi, err := buildimage.NewBuildImage(absKubeFile)
	if err != nil {
		return err
	}

	l.ImageNamed = named
	l.Context = absContext
	l.KubeFileName = absKubeFile
	l.BuildImage = bi
	return nil
}

func (l *Builder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	buildPipeline = append(buildPipeline,
		l.ExecBuild,
		l.SaveBuildImage,
		l.Cleanup,
	)
	return buildPipeline, nil
}

func (l *Builder) ExecBuild() error {
	ctx := buildimage.Context{
		BuildContext: l.Context,
		BuildType:    l.BuildType,
		UseCache:     !l.NoCache,
	}

	err := l.BuildImage.ExecBuild(ctx)
	if err != nil {
		return err
	}

	return l.checkPodStatus()
}

func (l *Builder) checkPodStatus() error {
	client, _ := k8s.Newk8sClient()
	if client == nil {
		return nil
	}

	if !l.IsAllPodsRunning(client) {
		return fmt.Errorf("cache docker image failed,cluster pod not running")
	}

	return nil
}
func (l *Builder) IsAllPodsRunning(k8sClient *k8s.Client) bool {
	logger.Info("waiting resource to sync")
	//wait resource to sync.do sleep here,because we can't fetch the pod status immediately.
	//if we use retry to check pod status, will pass the cache part, due to some resources has not been created yet.
	time.Sleep(30 * time.Second)
	err := utils.Retry(10, 5*time.Second, func() error {
		namespacePodList, err := k8sClient.ListAllNamespacesPods()
		if err != nil {
			return err
		}

		var notRunning int
		for _, podNamespace := range namespacePodList {
			for _, pod := range podNamespace.PodList.Items {
				if pod.Status.Phase != "Running" && pod.Status.Phase != "Succeeded" {
					logger.Info(podNamespace.Namespace.Name, pod.Name, pod.Status.Phase)
					notRunning++
					continue
				}
			}
		}
		if notRunning > 0 {
			logger.Info("remaining %d pod not running", notRunning)
			return fmt.Errorf("pod not running")
		}
		return nil
	})
	return err == nil
}

func (l *Builder) SaveBuildImage() error {
	imageName := l.ImageNamed.Raw()

	err := l.BuildImage.SaveBuildImage(imageName, buildimage.SaveOpts{
		WithoutBase: l.NoBase,
	})
	if err != nil {
		return err
	}
	logger.Info("update image %s to image metadata success !", imageName)
	return nil
}

func (l *Builder) Cleanup() (err error) {
	return l.BuildImage.Cleanup()
}
