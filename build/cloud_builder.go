package build

import (
	"fmt"
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"gitlab.alibaba-inc.com/seadent/pkg/image"
	"gitlab.alibaba-inc.com/seadent/pkg/infra"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	"gitlab.alibaba-inc.com/seadent/pkg/utils"
	"gitlab.alibaba-inc.com/seadent/pkg/utils/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
)

// cloud builder using cloud provider to build a cluster image
type CloudBuilder struct {
	local        *LocalBuilder
	RemoteHostIp string
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
	if c.local.IsOnlyCopy() {
		buildPipeline = append(buildPipeline,
			c.local.ExecBuild,
			c.local.UpdateImageMetadata,
			c.local.PushToRegistry)
	} else {
		buildPipeline = append(buildPipeline,
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

// load cluster file from disk
func (c *CloudBuilder) InitClusterFile() error {
	clusterFile := common.TmpClusterfile
	if !utils.IsFileExist(clusterFile) {
		err := c.getClusterFile()
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

func (c *CloudBuilder) getClusterFile() error {
	// find cluster file from context
	if c.getClusterFileFromContext() {
		logger.Info("get cluster file from context success!")
		return nil
	}
	// find cluster file from base image
	clusterFile := image.GetClusterFileFromImage(c.local.Image.Spec.Layers[0].Value)
	if clusterFile == "" {
		return fmt.Errorf("failed to find cluster file")
	}

	err := utils.WriteFile(common.RawClusterfile, []byte(clusterFile))
	if err != nil {
		return fmt.Errorf("failed to write cluster file:%v", err)
	}
	return nil
}

func (c *CloudBuilder) getClusterFileFromContext() bool {
	for i := range c.local.Image.Spec.Layers {
		layer := c.local.Image.Spec.Layers[i]
		if layer.Type == common.COPYCOMMAND && strings.Fields(layer.Value)[0] == common.DefaultClusterFileName {
			if _, err := utils.CopySingleFile(strings.Fields(layer.Value)[0], common.RawClusterfile); err != nil {
				return false
			}
			return true
		}
	}
	return false
}

// apply infra create vms
func (c *CloudBuilder) ApplyInfra() error {
	if c.local.Cluster.Spec.Provider == common.ALI_CLOUD {
		infraManager := infra.NewDefaultProvider(c.local.Cluster)
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
	c.SSH = client.Ssh
	c.RemoteHostIp = client.Host
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
	c.local.Cluster.Spec.Provider = common.ALI_CLOUD
	infraManager := infra.NewDefaultProvider(c.local.Cluster)
	if err := infraManager.Apply(); err != nil {
		logger.Info("failed to cleanup infra :%v", err)
	}

	tarFileName := fmt.Sprintf(common.TmpTarFile, c.local.Image.Spec.ID)
	if err = os.Remove(tarFileName); err != nil {
		logger.Info("failed to cleanup local temp file %s:%v", tarFileName, err)
	}
	if err = os.Remove(common.TmpClusterfile); err != nil {
		logger.Info("failed to cleanup local temp file %s:%v", common.TmpClusterfile, err)
	}
	if err = os.Remove(common.RawClusterfile); err != nil {
		logger.Info("failed to cleanup local temp file %s:%v", common.RawClusterfile, err)
	}
	logger.Info("cleanup success !")
	return nil
}

func NewCloudBuilder(cloudConfig *Config) Interface {
	local := new(LocalBuilder)
	local.Config = cloudConfig
	return &CloudBuilder{
		local: local,
	}
}
