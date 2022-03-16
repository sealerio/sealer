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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/utils/archive"

	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/utils/mount"

	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/build/buildkit/buildimage"
	"github.com/alibaba/sealer/pkg/image/reference"

	"github.com/alibaba/sealer/build/buildkit"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/checker"
	"github.com/alibaba/sealer/pkg/infra"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
)

const (
	RegistryMountUpper = "/var/lib/sealer/tmp/upper"
	RegistryMountWork  = "/var/lib/sealer/tmp/work"
)

var providerMap = map[string]string{
	common.LocalBuild:     common.BAREMETAL,
	common.AliCloudBuild:  common.AliCloud,
	common.ContainerBuild: common.CONTAINER,
}

// Builder using cloud provider to build a cluster image
type cloudBuilder struct {
	buildType          string
	noCache            bool
	noBase             bool
	provider           string
	tmpClusterFilePath string
	imageNamed         reference.Named
	context            string
	kubeFileName       string
	remoteHostIP       string
	buildArgs          map[string]string
	SSHClient          ssh.Interface
	cluster            *v1.Cluster
	image              *v1.Image
}

func (c *cloudBuilder) Build(name string, context string, kubefileName string) error {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return err
	}
	c.imageNamed = named

	absContext, absKubeFile, err := buildkit.ParseBuildArgs(context, kubefileName)
	if err != nil {
		return err
	}
	c.kubeFileName = absKubeFile

	err = buildkit.ValidateContextDirectory(absContext)
	if err != nil {
		return err
	}
	c.context = absContext

	image, err := buildimage.InitImageSpec(absKubeFile)
	if err != nil {
		return err
	}
	c.image = image

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

func (c *cloudBuilder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error

	buildPipeline = append(buildPipeline,
		c.PreCheck,
		c.InitClusterFile,
		c.ApplyInfra,
		c.SendBuildContext,
		c.RemoteLocalBuild,
		c.Cleanup,
	)
	return buildPipeline, nil
}

// PreCheck : check env before run cloud build
func (c *cloudBuilder) PreCheck() (err error) {
	if c.provider != common.AliCloud {
		return nil
	}
	registryChecker := checker.NewRegistryChecker(c.imageNamed.Domain())
	return registryChecker.Check(nil, checker.PhasePre)
}

// InitClusterFile load cluster file from disk
func (c *cloudBuilder) InitClusterFile() error {
	var cluster v1.Cluster
	if utils.IsFileExist(c.tmpClusterFilePath) {
		err := utils.UnmarshalYamlFile(c.tmpClusterFilePath, &cluster)
		if err != nil {
			return fmt.Errorf("failed to read %s:%v", c.tmpClusterFilePath, err)
		}
		c.cluster = &cluster
		return nil
	}

	rawClusterFile, err := buildimage.GetRawClusterFile(c.image.Spec.Layers[0].Value, c.image.Spec.Layers)
	if err != nil {
		return fmt.Errorf("failed to get base image err: %s", err)
	}

	if err := yaml.Unmarshal([]byte(rawClusterFile), &cluster); err != nil {
		return err
	}

	cluster.Spec.Provider = c.provider
	c.cluster = &cluster
	logger.Info("init cluster file success, provider type is %s", c.provider)
	return nil
}

// ApplyInfra apply infra create vms
func (c *cloudBuilder) ApplyInfra() (err error) {
	//bare_metal: no need to apply infra
	//ali_cloud,container: apply infra as cluster content
	if c.cluster.Spec.Provider == common.BAREMETAL {
		return c.initBuildSSH()
	}
	infraManager, err := infra.NewDefaultProvider(c.cluster)
	if err != nil {
		return err
	}
	if err := infraManager.Apply(); err != nil {
		return fmt.Errorf("failed to apply infra :%v", err)
	}

	c.cluster.Spec.Provider = common.BAREMETAL
	if err := utils.MarshalYamlToFile(c.tmpClusterFilePath, c.cluster); err != nil {
		return fmt.Errorf("failed to write cluster info:%v", err)
	}
	logger.Info("apply infra success !")
	return c.initBuildSSH()
}

func (c *cloudBuilder) initBuildSSH() error {
	// init ssh client
	c.cluster.Spec.Provider = c.provider
	client, err := ssh.NewSSHClientWithCluster(c.cluster)
	if err != nil {
		return fmt.Errorf("failed to prepare cluster ssh client:%v", err)
	}
	c.SSHClient = client.SSH
	c.remoteHostIP = client.Host
	return nil
}

// SendBuildContext send build context dir to remote host
func (c *cloudBuilder) SendBuildContext() error {
	if err := c.sendBuildContext(); err != nil {
		return fmt.Errorf("failed to send context: %v", err)
	}
	// change local builder context to ".", because sendBuildContext will send current localBuilder.Context to remote
	// and work within the localBuilder.Context remotely, so change context to "." is more appropriate.
	c.changeBuilderContext()
	return nil
}

// RemoteLocalBuild run sealer build remotely
func (c *cloudBuilder) RemoteLocalBuild() (err error) {
	// apply k8s cluster first
	apply := fmt.Sprintf("%s apply -f %s", common.RemoteSealerPath, c.tmpClusterFilePath)
	err = c.SSHClient.CmdAsync(c.remoteHostIP, apply)
	if err != nil {
		return fmt.Errorf("failed to run remote apply:%v", err)
	}
	// prepare registry
	if err = c.prepareRegistry(); err != nil {
		return err
	}
	// run sealer build cmd
	return c.runBuildCommands()
}

// prepareRegistry: collect operator images via remount registry.
// because runtime apply registry not mount the rootfs/registry as lower layer. if we want to collect
// operator images,we must remount the dir rootfs/registry for collecting differ of the overlay upper layer.
func (c *cloudBuilder) prepareRegistry() (err error) {
	rootfs := common.DefaultTheClusterRootfsDir(c.cluster.Name)
	mkdir := fmt.Sprintf("rm -rf %s %s && mkdir -p %s %s", RegistryMountUpper, RegistryMountWork,
		RegistryMountUpper, RegistryMountWork)

	mountCmd := fmt.Sprintf("%s && mount -t overlay overlay -o lowerdir=%s,upperdir=%s,workdir=%s %s", mkdir,
		rootfs,
		RegistryMountUpper, RegistryMountWork, rootfs)
	isMount, _ := mount.GetRemoteMountDetails(c.SSHClient, c.remoteHostIP, rootfs)
	if isMount {
		mountCmd = fmt.Sprintf("umount %s && %s", rootfs, mountCmd)
	}
	// we need to restart registry container in order to cache the container images.
	mountCmd = fmt.Sprintf("%s && docker restart %s", mountCmd, runtime.RegistryName)

	if err := c.SSHClient.CmdAsync(c.remoteHostIP, mountCmd); err != nil {
		return err
	}
	return nil
}

func (c *cloudBuilder) runBuildCommands() (err error) {
	// run local build command
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.cluster.Name)
	build := fmt.Sprintf(common.BuildClusterCmd, common.RemoteSealerPath,
		filepath.Base(c.kubeFileName), c.imageNamed.Raw(), common.LocalBuild, ".")
	if c.noBase {
		build = fmt.Sprintf("%s %s", build, "--base=false")
	}
	if c.noCache {
		build = fmt.Sprintf("%s %s", build, "--no-cache=true")
	}
	if len(c.buildArgs) != 0 {
		arg := strings.Join(utils.ConvertMapToEnvList(c.buildArgs), " ")
		build = fmt.Sprintf("%s %s %s", build, "--build-arg", arg)
	}
	if c.provider == common.AliCloud {
		push := fmt.Sprintf(common.PushImageCmd, common.RemoteSealerPath,
			c.imageNamed.Raw())
		build = fmt.Sprintf("%s && %s", build, push)
	}
	logger.Info("run remote shell %s", build)

	cmd := fmt.Sprintf("cd %s && %s", workdir, build)
	return c.SSHClient.CmdAsync(c.remoteHostIP, cmd)
}

// Cleanup cleanup infra and tmp file
func (c *cloudBuilder) Cleanup() (err error) {
	t := metav1.Now()
	c.cluster.DeletionTimestamp = &t
	c.cluster.Spec.Provider = c.provider
	infraManager, err := infra.NewDefaultProvider(c.cluster)
	if err != nil {
		return err
	}
	if err := infraManager.Apply(); err != nil {
		logger.Info("failed to cleanup infra :%v", err)
	}

	if err = os.Remove(c.tmpClusterFilePath); err != nil {
		logger.Warn("failed to cleanup local temp file %s:%v", c.tmpClusterFilePath, err)
	}

	logger.Info("cleanup success !")
	return nil
}

//sendBuildContext:send local build context to remote server
func (c *cloudBuilder) sendBuildContext() (err error) {
	// if remote cluster already exist,no need to pre init master0
	if !c.SSHClient.IsFileExist(c.remoteHostIP, common.RemoteSealerPath) {
		err = runtime.PreInitMaster0(c.SSHClient, c.remoteHostIP)
		if err != nil {
			return fmt.Errorf("failed to prepare cluster env %v", err)
		}
	}
	tarFileName := fmt.Sprintf(common.TmpTarFile, utils.GenUniqueID(32))
	err = tarBuildContext(c.kubeFileName, c.context, tarFileName)
	if err != nil {
		return err
	}
	defer func() {
		if err = os.Remove(tarFileName); err != nil {
			logger.Warn("failed to cleanup local temp file %s:%v", tarFileName, err)
		}
	}()
	// send to remote server
	workdir := fmt.Sprintf(common.DefaultWorkDir, c.cluster.Name)
	if err = c.SSHClient.Copy(c.remoteHostIP, tarFileName, tarFileName); err != nil {
		return fmt.Errorf("failed to copy tar file: %s, err: %v", tarFileName, err)
	}
	// unzip remote context
	err = c.SSHClient.CmdAsync(c.remoteHostIP, fmt.Sprintf(common.UnzipCmd, workdir, tarFileName, workdir))
	if err != nil {
		return err
	}
	logger.Info("send build context to %s success !", c.remoteHostIP)
	return nil
}

func (c *cloudBuilder) changeBuilderContext() {
	c.context = "."
}

func tarBuildContext(kubeFilePath string, context string, tarFileName string) error {
	file, err := os.Create(filepath.Clean(tarFileName))
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", tarFileName, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("failed to close file")
		}
	}()

	var pathsToCompress []string
	pathsToCompress = append(pathsToCompress, kubeFilePath, context)
	tarReader, err := archive.TarWithoutRootDir(pathsToCompress...)
	if err != nil {
		return fmt.Errorf("failed to new tar reader when send build context, err: %v", err)
	}
	defer tarReader.Close()

	_, err = io.Copy(file, tarReader)
	if err != nil {
		return fmt.Errorf("failed to tar build context, err: %v", err)
	}
	return nil
}

func NewCloudBuilder(config *Config) (Interface, error) {
	provider := common.AliCloud
	if config.BuildType != "" {
		provider = providerMap[config.BuildType]
	}

	return &cloudBuilder{
		buildType:          config.BuildType,
		noCache:            config.NoCache,
		noBase:             config.NoBase,
		buildArgs:          config.BuildArgs,
		provider:           provider,
		tmpClusterFilePath: common.TmpClusterfile,
	}, nil
}
