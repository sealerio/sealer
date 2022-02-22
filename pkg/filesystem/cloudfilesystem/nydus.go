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

package cloudfilesystem

import (
	"context"
	"fmt"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/env"

	"github.com/alibaba/sealer/pkg/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
)

const (
	RemoteNydusdInit = "cd %s && chmod +x *.sh && bash start.sh %s"
	RemoteNydusdStop = "sh %s && rm -rf %s"
)

type nydusFileSystem struct {
}

func (n *nydusFileSystem) MountRootfs(cluster *v2.Cluster, hosts []string, initFlag bool) error {
	var (
		clusterRootfsDir = common.DefaultTheClusterRootfsDir(cluster.Name)
		src              = filepath.Join(common.DefaultMountCloudImageDir(cluster.Name), "nydusdfile")
		dest             = common.DefaultTheClusterNydusdFileDir(cluster.Name)
		nydusdCpCmd      = fmt.Sprintf("rm -rf %s && cp -r %s %s", dest, src, dest)
	)
	//cp "nydusdfile" dir from mounted dir to rootfs.
	_, err := utils.RunSimpleCmd(nydusdCpCmd)
	if err != nil {
		return fmt.Errorf("cp nydusdfile failed %v", err)
	}
	//scp roofs to all Masters and Nodes,then do init.sh
	if err := mountNydusRootfs(hosts, clusterRootfsDir, cluster, initFlag); err != nil {
		return fmt.Errorf("mount rootfs failed %v", err)
	}

	return nil
}

func (n *nydusFileSystem) UnMountRootfs(cluster *v2.Cluster, hosts []string) error {
	var (
		nydusdDir            = common.DefaultTheClusterNydusdDir(cluster.Name)
		nydusdFileDir        = common.DefaultTheClusterNydusdFileDir(cluster.Name)
		nydusdServerCleanCmd = filepath.Join(nydusdFileDir, "serverclean.sh")
	)
	//do clean.sh,then remove all Masters and Nodes roofs
	if err := unmountRootfs(hosts, cluster); err != nil {
		return err
	}

	if utils.IsExist(nydusdServerCleanCmd) {
		cleanCmd := fmt.Sprintf("sh %s && rm -rf %s", nydusdServerCleanCmd, nydusdDir)
		_, err := utils.RunSimpleCmd(cleanCmd)
		if err != nil {
			return fmt.Errorf("failed to stop nydusdserver %v", err)
		}
	}
	return nil
}

func mountNydusRootfs(ipList []string, target string, cluster *v2.Cluster, initFlag bool) error {
	var (
		src               = common.DefaultMountCloudImageDir(cluster.Name)
		nydusdDir         = common.DefaultTheClusterNydusdDir(cluster.Name)
		nydusdFileDir     = common.DefaultTheClusterNydusdFileDir(cluster.Name)
		startNydusdServer = fmt.Sprintf("cd %s && chmod +x serverstart.sh && ./serverstart.sh %s %s", nydusdFileDir, src, nydusdDir)
		nydusdInitCmd     = fmt.Sprintf(RemoteNydusdInit, nydusdDir, target)
		nydusdCleanCmd    = fmt.Sprintf(RemoteNydusdStop, filepath.Join(nydusdDir, "clean.sh"), nydusdDir)
		cleanCmd          = fmt.Sprintf("echo '%s' >> "+common.DefaultClusterClearBashFile, nydusdCleanCmd, cluster.Name)
		envProcessor      = env.NewEnvProcessor(cluster)
		config            = runtime.GetRegistryConfig(src, runtime.GetMaster0Ip(cluster))
		initCmd           = fmt.Sprintf(RemoteChmod, target, config.Domain, config.Port)
	)

	// use env list to render image mount dir: etc,charts,manifests.
	err := renderENV(src, ipList, envProcessor)
	if err != nil {
		return err
	}

	//convert image and start nydusd http server
	_, err = utils.RunSimpleCmd(startNydusdServer)
	if err != nil {
		return fmt.Errorf("nydusdserver start fail %v", err)
	}
	logger.Info("nydus images converted and nydusd http server started")

	eg, _ := errgroup.WithContext(context.Background())
	for _, IP := range ipList {
		ip := IP
		eg.Go(func() error {
			sshClient, err := ssh.GetHostSSHClient(ip, cluster)
			if err != nil {
				return fmt.Errorf("get host ssh client failed %v", err)
			}
			err = copyFiles(sshClient, ip == config.IP, ip, nydusdDir, nydusdDir)
			if err != nil {
				return fmt.Errorf("scp nydusd failed %v", err)
			}
			if initFlag {
				err = sshClient.CmdAsync(ip, envProcessor.WrapperShell(ip, nydusdInitCmd))
				if err != nil {
					return fmt.Errorf("init nydusd failed %v", err)
				}
				err = sshClient.CmdAsync(ip, envProcessor.WrapperShell(ip, initCmd))
				if err != nil {
					return fmt.Errorf("exec init.sh failed %v", err)
				}
				err = sshClient.CmdAsync(ip, envProcessor.WrapperShell(ip, cleanCmd))
				if err != nil {
					return fmt.Errorf("echo nydusdcleancmd to clean.sh failed %v", err)
				}
			}
			return err
		})
	}
	return eg.Wait()
}

func NewNydusFileSystem() (Interface, error) {
	return &nydusFileSystem{}, nil
}
