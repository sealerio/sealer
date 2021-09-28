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

package cloud

import (
	"fmt"

	"os"
	"path/filepath"

	"github.com/alibaba/sealer/build/local"

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

// Builder using cloud provider to build a cluster image
type Builder struct {
	Local              *local.Builder
	RemoteHostIP       string
	SSH                ssh.Interface
	Provider           string
	TmpClusterFilePath string
}

func (c *Builder) Build(name string, context string, kubefileName string) error {
	err := c.Local.InitBuilder(name, context, kubefileName)
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

func (c *Builder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	if err := c.Local.InitImageSpec(); err != nil {
		return nil, err
	}
	if local.IsOnlyCopy(c.Local.Image.Spec.Layers) {
		buildPipeline = append(buildPipeline,
			c.Local.PullBaseImageNotExist,
			c.Local.ExecBuild,
			c.Local.UpdateImageMetadata)
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
func (c *Builder) PreCheck() (err error) {
	if c.Provider != common.AliCloud {
		return nil
	}
	registryChecker := checker.NewRegistryChecker(c.Local.ImageNamed.Domain())
	return registryChecker.Check()
}

// load cluster file from disk
func (c *Builder) InitClusterFile() error {
	var cluster v1.Cluster
	if utils.IsFileExist(c.TmpClusterFilePath) {
		err := utils.UnmarshalYamlFile(c.TmpClusterFilePath, &cluster)
		if err != nil {
			return fmt.Errorf("failed to read %s:%v", c.TmpClusterFilePath, err)
		}
		c.Local.Cluster = &cluster
		return nil
	}

	rawClusterFile, err := local.GetRawClusterFile(c.Local.Image)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal([]byte(rawClusterFile), &cluster); err != nil {
		return err
	}

	cluster.Spec.Provider = c.Provider
	c.Local.Cluster = &cluster
	logger.Info("init cluster file success, provider type is %s", c.Provider)
	return nil
}

// apply infra create vms
func (c *Builder) ApplyInfra() (err error) {
	//bare_metal: no need to apply infra
	//ali_cloud,container: apply infra as cluster content
	if c.Local.Cluster.Spec.Provider == common.BAREMETAL {
		return c.initBuildSSH()
	}
	infraManager, err := infra.NewDefaultProvider(c.Local.Cluster)
	if err != nil {
		return err
	}
	if err := infraManager.Apply(); err != nil {
		return fmt.Errorf("failed to apply infra :%v", err)
	}

	c.Local.Cluster.Spec.Provider = common.BAREMETAL
	if err := utils.MarshalYamlToFile(c.TmpClusterFilePath, c.Local.Cluster); err != nil {
		return fmt.Errorf("failed to write cluster info:%v", err)
	}
	logger.Info("apply infra success !")
	return c.initBuildSSH()
}

func (c *Builder) initBuildSSH() error {
	// init ssh client
	c.Local.Cluster.Spec.Provider = c.Provider
	client, err := ssh.NewSSHClientWithCluster(c.Local.Cluster)
	if err != nil {
		return fmt.Errorf("failed to prepare cluster ssh client:%v", err)
	}
	c.SSH = client.SSH
	c.RemoteHostIP = client.Host
	return nil
}

// send build context dir to remote host
func (c *Builder) SendBuildContext() error {
	if err := c.sendBuildContext(); err != nil {
		return fmt.Errorf("failed to send context: %v", err)
	}
	// change local builder context to ".", because sendBuildContext will send current localBuilder.Context to remote
	// and work within the localBuilder.Context remotely, so change context to "." is more appropriate.
	c.changeBuilderContext()
	return nil
}

// run sealer build remotely
func (c *Builder) RemoteLocalBuild() (err error) {
	// apply k8s cluster first
	apply := fmt.Sprintf("%s apply -f %s", common.RemoteSealerPath, c.TmpClusterFilePath)
	err = c.SSH.CmdAsync(c.RemoteHostIP, apply)
	if err != nil {
		return fmt.Errorf("failed to run remote apply:%v", err)
	}
	return c.runBuildCommands()
}

func (c *Builder) runBuildCommands() (err error) {
	// run local build command
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.Local.Cluster.Name)
	build := fmt.Sprintf(common.BuildClusterCmd, common.RemoteSealerPath,
		filepath.Base(c.Local.KubeFileName), c.Local.ImageNamed.Raw(), common.LocalBuild, ".")

	if c.Provider == common.AliCloud {
		push := fmt.Sprintf(common.PushImageCmd, common.RemoteSealerPath,
			c.Local.ImageNamed.Raw())
		build = fmt.Sprintf("%s && %s", build, push)
	}
	logger.Info("run remote shell %s", build)

	cmd := fmt.Sprintf("cd %s && %s", workdir, build)
	return c.SSH.CmdAsync(c.RemoteHostIP, cmd)
}

//cleanup infra and tmp file
func (c *Builder) Cleanup() (err error) {
	t := metav1.Now()
	c.Local.Cluster.DeletionTimestamp = &t
	c.Local.Cluster.Spec.Provider = c.Provider
	infraManager, err := infra.NewDefaultProvider(c.Local.Cluster)
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
