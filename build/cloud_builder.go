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

	"github.com/alibaba/sealer/check/checker"

	"github.com/alibaba/sealer/image/store"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/infra"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// cloud builder using cloud provider to build a cluster image
type CloudBuilder struct {
	local        *LocalBuilder
	RemoteHostIP string
	SSH          ssh.Interface
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
	if c.IsOnlyCopy() {
		buildPipeline = append(buildPipeline,
			c.local.PullBaseImageNotExist,
			c.local.ExecBuild,
			c.local.UpdateImageMetadata)
	} else {
		buildPipeline = append(buildPipeline,
			c.PreCheck,
			c.InitClusterFile,
			c.ApplyInfra,
			c.InitBuildSSH,
			c.SendBuildContext,
			c.RemoteLocalBuild,
			c.Cleanup,
		)
	}
	return buildPipeline, nil
}

// PreCheck: check env before run cloud build
func (c *CloudBuilder) PreCheck() error {
	registryChecker := checker.NewRegistryChecker(c.local.ImageNamed.Domain())
	err := registryChecker.Check()
	if err != nil {
		return err
	}
	return nil
}

func (c *CloudBuilder) IsOnlyCopy() bool {
	for i := 1; i < len(c.local.Image.Spec.Layers); i++ {
		if c.local.Image.Spec.Layers[i].Type == common.RUNCOMMAND ||
			c.local.Image.Spec.Layers[i].Type == common.CMDCOMMAND {
			return false
		}
	}
	return true
}

// load cluster file from disk
func (c *CloudBuilder) InitClusterFile() error {
	clusterFile := common.TmpClusterfile
	if !utils.IsFileExist(clusterFile) {
		rawClusterFile := GetRawClusterFile(c.local.Image)
		if rawClusterFile == "" {
			return fmt.Errorf("failed to get cluster file from context or base image")
		}
		err := utils.WriteFile(common.RawClusterfile, []byte(rawClusterFile))
		if err != nil {
			return err
		}
		clusterFile = common.RawClusterfile
	}
	var cluster v1.Cluster
	err := utils.UnmarshalYamlFile(clusterFile, &cluster)
	if err != nil {
		return fmt.Errorf("failed to read %s:%v", clusterFile, err)
	}
	c.local.Cluster = &cluster

	logger.Info("read cluster file %s success !", clusterFile)
	return nil
}

// apply infra create vms
func (c *CloudBuilder) ApplyInfra() error {
	if c.local.Cluster.Spec.Provider == common.AliCloud {
		infraManager, err := infra.NewDefaultProvider(c.local.Cluster)
		if err != nil {
			return err
		}
		if err := infraManager.Apply(); err != nil {
			return fmt.Errorf("failed to apply infra :%v", err)
		}
		c.local.Cluster.Spec.Provider = common.BAREMETAL
		if err := utils.MarshalYamlToFile(common.TmpClusterfile, c.local.Cluster); err != nil {
			return fmt.Errorf("failed to write cluster info:%v", err)
		}
	}
	logger.Info("apply infra success !")
	return nil
}
func (c *CloudBuilder) InitBuildSSH() error {
	// init ssh client
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
	return c.sendBuildContext()
}

// run BUILD CMD commands
func (c *CloudBuilder) RemoteLocalBuild() (err error) {
	return c.runBuildCommands()
}

//cleanup infra and tmp file
func (c *CloudBuilder) Cleanup() (err error) {
	t := metav1.Now()
	c.local.Cluster.DeletionTimestamp = &t
	c.local.Cluster.Spec.Provider = common.AliCloud
	infraManager, err := infra.NewDefaultProvider(c.local.Cluster)
	if err != nil {
		return err
	}
	if err := infraManager.Apply(); err != nil {
		logger.Info("failed to cleanup infra :%v", err)
	}

	if err = os.Remove(common.TmpClusterfile); err != nil {
		logger.Warn("failed to cleanup local temp file %s:%v", common.TmpClusterfile, err)
	}
	if err = os.Remove(common.RawClusterfile); err != nil {
		logger.Info("failed to cleanup local temp file %s:%v", common.RawClusterfile, err)
	}
	logger.Info("cleanup success !")
	return nil
}

func NewCloudBuilder(cloudConfig *Config) (Interface, error) {
	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, err
	}
	return &CloudBuilder{
		local: &LocalBuilder{
			Config:     cloudConfig,
			LayerStore: layerStore,
		},
	}, nil
}
