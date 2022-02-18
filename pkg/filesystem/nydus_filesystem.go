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

package filesystem

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

type NydusFileSystem struct {
	*FileSystem
}

func (c *NydusFileSystem) MountRootfs(cluster *v2.Cluster, hosts []string, initFlag bool) error {
	clusterRootfsDir := common.DefaultTheClusterRootfsDir(cluster.Name)
	//scp roofs to all Masters and Nodes,then do init.sh
	md, err := runtime.LoadMetadata(common.DefaultMountCloudImageDir(cluster.Name))
	if err != nil || md == nil {
		return fmt.Errorf("LoadMetadata failed %v", err)
	}
	if md.NydusFlag {
		mountnydusdfile := filepath.Join(common.DefaultMountCloudImageDir(cluster.Name), "nydusdfile")
		clusternydusdfile := common.DefaultTheClusterNydusdFileDir(cluster.Name)
		nydusdcp := fmt.Sprintf("rm -rf %s && cp -r %s %s", clusternydusdfile, mountnydusdfile, clusternydusdfile)
		_, err := utils.RunSimpleCmd(nydusdcp)
		if err != nil {
			return fmt.Errorf("cp nydusdfile failed %v", err)
		}
		if err := mountNydusRootfs(hosts, clusterRootfsDir, cluster, initFlag); err != nil {
			return fmt.Errorf("mount rootfs failed %v", err)
		}
	} else {
		if err := mountRootfs(hosts, clusterRootfsDir, cluster, initFlag); err != nil {
			return fmt.Errorf("mount rootfs failed %v", err)
		}
	}
	return nil
}

func (c *NydusFileSystem) UnMountRootfs(cluster *v2.Cluster, hosts []string) error {
	//do clean.sh,then remove all Masters and Nodes roofs
	if err := unmountRootfs(hosts, cluster); err != nil {
		return err
	}
	nydusdserverclean := filepath.Join(common.DefaultTheClusterNydusdFileDir(cluster.Name), "serverclean.sh")
	if utils.IsExist(nydusdserverclean) {
		cleancmd := fmt.Sprintf("sh %s && rm -rf %s", nydusdserverclean, common.DefaultTheClusterNydusdDir(cluster.Name))
		_, err := utils.RunSimpleCmd(cleancmd)
		if err != nil {
			return fmt.Errorf("stop nydusdserver failed %v", err)
		}
	}
	return nil
}

const (
	RemoteNydusdInit = "cd %s && chmod +x *.sh && bash start.sh %s"
	RemoteNydusdStop = "sh %s && rm -rf %s"
)

func mountNydusRootfs(ipList []string, target string, cluster *v2.Cluster, initFlag bool) error {
	/*	if err := ssh.WaitSSHToReady(*cluster, 6, ipList...); err != nil {
		return errors.Wrap(err, "check for node ssh service time out")
	}*/
	config := runtime.GetRegistryConfig(
		common.DefaultTheClusterRootfsDir(cluster.Name),
		runtime.GetMaster0Ip(cluster))
	src := common.DefaultMountCloudImageDir(cluster.Name)
	renderEtc := filepath.Join(src, common.EtcDir)
	renderChart := filepath.Join(src, common.RenderChartsDir)
	renderManifests := filepath.Join(src, common.RenderManifestsDir)

	nydusdsrc := common.DefaultTheClusterNydusdDir(cluster.Name)
	scptarget := common.DefaultTheClusterNydusdDir(cluster.Name)
	nydusdfiledir := common.DefaultTheClusterNydusdFileDir(cluster.Name)
	//convert image and start nydusd_http_server
	nydusdservercmd := fmt.Sprintf("cd %s && chmod +x serverstart.sh && ./serverstart.sh %s %s", nydusdfiledir, src, nydusdsrc)
	_, err := utils.RunSimpleCmd(nydusdservercmd)
	if err != nil {
		return fmt.Errorf("nydusdserver start fail %v", err)
	}
	logger.Info("nydus images converted and nydusd_http_server started!")
	nydusdCmd := fmt.Sprintf(RemoteNydusdInit, scptarget, target)
	nydusdclean := fmt.Sprintf(RemoteNydusdStop, filepath.Join(scptarget, "clean.sh"), scptarget)
	echocleancmd := fmt.Sprintf("echo '%s' >> "+common.DefaultClusterClearBashFile, nydusdclean, cluster.Name)
	// TODO scp sdk has change file mod bug
	initCmd := fmt.Sprintf(RemoteChmod, target)
	envProcessor := env.NewEnvProcessor(cluster)
	eg, _ := errgroup.WithContext(context.Background())

	for _, IP := range ipList {
		ip := IP
		eg.Go(func() error {
			for _, dir := range []string{renderEtc, renderChart, renderManifests} {
				if utils.IsExist(dir) {
					err := envProcessor.RenderAll(ip, dir)
					if err != nil {
						return err
					}
				}
			}
			sshClient, err := ssh.GetHostSSHClient(ip, cluster)
			if err != nil {
				return fmt.Errorf("get host ssh client failed %v", err)
			}
			err = CopyFiles(sshClient, ip == config.IP, ip, nydusdsrc, scptarget)
			if err != nil {
				return fmt.Errorf("scp nydusd failed %v", err)
			}
			if initFlag {
				err = sshClient.CmdAsync(ip, envProcessor.WrapperShell(ip, nydusdCmd))
				if err != nil {
					return fmt.Errorf("init nydusd failed %v", err)
				}
				err = sshClient.CmdAsync(ip, envProcessor.WrapperShell(ip, initCmd))
				if err != nil {
					return fmt.Errorf("exec init.sh failed %v", err)
				}
				err = sshClient.CmdAsync(ip, envProcessor.WrapperShell(ip, echocleancmd))
				if err != nil {
					return fmt.Errorf("echo nydusdcleancmd to clean.sh failed %v", err)
				}
			}
			return err
		})
	}
	return eg.Wait()
}
