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
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/check/checker"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/infra"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
)

var ProviderMap = map[string]string{
	common.LocalBuild:     common.BAREMETAL,
	common.AliCloudBuild:  common.AliCloud,
	common.ContainerBuild: common.CONTAINER,
}

// cloud builder using cloud provider to build a cluster image
type CloudBuilder struct {
	local              *LocalBuilder
	RemoteHostIP       string
	SSH                ssh.Interface
	Provider           string
	TmpClusterFilePath string
}

func (c *CloudBuilder) Build(name string, context string, kubefileName string) error {
	err := c.local.initBuilder(name, context, kubefileName)
	if err != nil {
		return err
	}

	pipLine, err := c.GetBuildPipeLine()
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

func (c *CloudBuilder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	if err := c.local.InitImageSpec(); err != nil {
		return nil, err
	}
	if IsOnlyCopy(c.local.Image.Spec.Layers) {
		buildPipeline = append(buildPipeline,
			c.local.PullBaseImageNotExist,
			c.local.ExecBuild,
			c.local.UpdateImageMetadata)
	} else {
		buildPipeline = append(buildPipeline,
			c.PreCheck,
			c.InitClusterFile,
			c.ApplyInfra,
			c.SendBuildContext,
			c.RemoteLocalBuild,
			c.Cleanup,
		)
	}
	return buildPipeline, nil
}

// PreCheck: check env before run cloud build
func (c *CloudBuilder) PreCheck() (err error) {
	if c.Provider != common.AliCloud {
		return nil
	}
	registryChecker := checker.NewRegistryChecker(c.local.ImageNamed.Domain())
	return registryChecker.Check()
}

// load cluster file from disk
func (c *CloudBuilder) InitClusterFile() error {
	var cluster v1.Cluster
	if utils.IsFileExist(c.TmpClusterFilePath) {
		err := utils.UnmarshalYamlFile(c.TmpClusterFilePath, &cluster)
		if err != nil {
			return fmt.Errorf("failed to read %s:%v", c.TmpClusterFilePath, err)
		}
		c.local.Cluster = &cluster
		return nil
	}

	rawClusterFile := GetRawClusterFile(c.local.Image)
	if rawClusterFile == "" {
		return fmt.Errorf("failed to get cluster file from context or base image")
	}
	err := yaml.Unmarshal([]byte(rawClusterFile), &cluster)
	if err != nil {
		return err
	}

	cluster.Spec.Provider = c.Provider
	c.local.Cluster = &cluster
	logger.Info("init cluster file success, provider type is %s", c.Provider)
	return nil
}

// apply infra create vms
func (c *CloudBuilder) ApplyInfra() (err error) {
	//bare_metal: no need to apply infra
	//ali_cloud,container: apply infra as cluster content
	if c.local.Cluster.Spec.Provider == common.BAREMETAL {
		return c.initBuildSSH()
	}
	infraManager, err := infra.NewDefaultProvider(c.local.Cluster)
	if err != nil {
		return err
	}
	if err := infraManager.Apply(); err != nil {
		return fmt.Errorf("failed to apply infra :%v", err)
	}

	c.local.Cluster.Spec.Provider = common.BAREMETAL
	if err := utils.MarshalYamlToFile(c.TmpClusterFilePath, c.local.Cluster); err != nil {
		return fmt.Errorf("failed to write cluster info:%v", err)
	}
	logger.Info("apply infra success !")
	return c.initBuildSSH()
}

func (c *CloudBuilder) initBuildSSH() error {
	// init ssh client
	c.local.Cluster.Spec.Provider = c.Provider
	client, err := ssh.NewSSHClientWithCluster(c.local.Cluster)
	if err != nil {
		return fmt.Errorf("failed to prepare cluster ssh client:%v", err)
	}
	c.SSH = client.SSH
	c.RemoteHostIP = client.Host
	return nil
}

// send build context dir to remote host
func (c *CloudBuilder) SendBuildContext() error {
	err := c.sendBuildContext()
	if err != nil {
		return fmt.Errorf("failed to send context")
	}
	// change local builder context to ".", because sendBuildContext will send current localBuilder.Context to remote
	// and work within the localBuilder.Context remotely, so change context to "." is more appropriate.
	c.changeBuilderContext()
	return nil
}

// run sealer build remotely
func (c *CloudBuilder) RemoteLocalBuild() (err error) {
	// apply k8s cluster first
	apply := fmt.Sprintf("%s apply -f %s", common.RemoteSealerPath, c.TmpClusterFilePath)
	err = c.SSH.CmdAsync(c.RemoteHostIP, apply)
	if err != nil {
		return fmt.Errorf("failed to run remote apply:%v", err)
	}
	return c.runBuildCommands()
}

func (c *CloudBuilder) runBuildCommands() (err error) {
	// run local build command
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.local.Cluster.Name)
	build := fmt.Sprintf(common.BuildClusterCmd, common.RemoteSealerPath,
		filepath.Base(c.local.KubeFileName), c.local.ImageNamed.Raw(), common.LocalBuild, ".")

	if c.Provider == common.AliCloud {
		push := fmt.Sprintf(common.PushImageCmd, common.RemoteSealerPath,
			c.local.ImageNamed.Raw())
		build = fmt.Sprintf("%s && %s", build, push)
	}
	logger.Info("run remote shell %s", build)

	cmd := fmt.Sprintf("cd %s && %s", workdir, build)
	return c.SSH.CmdAsync(c.RemoteHostIP, cmd)
}

//cleanup infra and tmp file
func (c *CloudBuilder) Cleanup() (err error) {
	t := metav1.Now()
	c.local.Cluster.DeletionTimestamp = &t
	c.local.Cluster.Spec.Provider = c.Provider
	infraManager, err := infra.NewDefaultProvider(c.local.Cluster)
	if err != nil {
		return err
	}
	if err := infraManager.Apply(); err != nil {
		logger.Info("failed to cleanup infra :%v", err)
	}

	if err = os.Remove(c.TmpClusterFilePath); err != nil {
		logger.Warn("failed to cleanup local temp file %s:%v", c.TmpClusterFilePath, err)
	}

	logger.Info("cleanup success !")
	return nil
}

func NewCloudBuilder(cloudConfig *Config) (Interface, error) {
	localBuilder, err := NewLocalBuilder(cloudConfig)
	if err != nil {
		return nil, err
	}

	provider := common.AliCloud
	if cloudConfig.BuildType != "" {
		provider = ProviderMap[cloudConfig.BuildType]
	}

	return &CloudBuilder{
		local:              localBuilder.(*LocalBuilder),
		Provider:           provider,
		TmpClusterFilePath: common.TmpClusterfile,
	}, nil
}
