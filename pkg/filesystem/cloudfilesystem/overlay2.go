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
	"sync"

	"github.com/alibaba/sealer/utils/platform"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/env"
	"github.com/alibaba/sealer/pkg/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils/mount"
	"github.com/alibaba/sealer/utils/ssh"
	"golang.org/x/sync/errgroup"
)

const (
	RemoteChmod = "cd %s  && chmod +x scripts/* && cd scripts && bash init.sh /var/lib/docker %s %s"
)

type overlayFileSystem struct {
}

func (o *overlayFileSystem) MountRootfs(cluster *v2.Cluster, hosts []string, initFlag bool) error {
	clusterRootfsDir := common.DefaultTheClusterRootfsDir(cluster.Name)
	//scp roofs to all Masters and Nodes,then do init.sh
	if err := mountRootfs(hosts, clusterRootfsDir, cluster, initFlag); err != nil {
		return fmt.Errorf("mount rootfs failed %v", err)
	}
	return nil
}

func (o *overlayFileSystem) UnMountRootfs(cluster *v2.Cluster, hosts []string) error {
	//do clean.sh,then remove all Masters and Nodes roofs
	if err := unmountRootfs(hosts, cluster); err != nil {
		return err
	}
	return nil
}

func mountRootfs(ipList []string, target string, cluster *v2.Cluster, initFlag bool) error {
	var (
		mountDirs            = make(map[string]bool)
		rwmt                 = &sync.RWMutex{}
		envProcessor         = env.NewEnvProcessor(cluster)
		clusterPlatform, err = ssh.GetClusterPlatform(cluster)
	)
	if err != nil {
		return err
	}
	eg, _ := errgroup.WithContext(context.Background())
	for _, IP := range ipList {
		ip := IP
		eg.Go(func() error {
			src := platform.GetMountCloudImagePlatformDir(cluster.Name, clusterPlatform[ip])
			config := runtime.GetRegistryConfig(src, runtime.GetMaster0Ip(cluster))
			initCmd := fmt.Sprintf(RemoteChmod, target, config.Domain, config.Port)
			rwmt.Lock()
			if !mountDirs[src] {
				// use env list to render image mount dir: etc,charts,manifests.
				mountDirs[src] = true
				rwmt.Unlock()
				err = renderENV(src, ipList, envProcessor)
				if err != nil {
					return err
				}
				sshClient, err := ssh.GetHostSSHClient(config.IP, cluster)
				if err != nil {
					return err
				}
				err = sshClient.Copy(config.IP, filepath.Join(src, "registry"), filepath.Join(target, "registry"))
				if err != nil {
					return err
				}
			}
			sshClient, err := ssh.GetHostSSHClient(ip, cluster)
			if err != nil {
				return fmt.Errorf("get host ssh client failed %v", err)
			}
			err = copyFiles(sshClient, false, ip, src, target)
			if err != nil {
				return fmt.Errorf("copy rootfs failed %v", err)
			}
			if initFlag {
				err = sshClient.CmdAsync(ip, envProcessor.WrapperShell(ip, initCmd))
				if err != nil {
					return fmt.Errorf("exec init.sh failed %v", err)
				}
			}
			return err
		})
	}
	return eg.Wait()
}

func unmountRootfs(ipList []string, cluster *v2.Cluster) error {
	var (
		clusterRootfsDir = common.DefaultTheClusterRootfsDir(cluster.Name)
		cleanFile        = fmt.Sprintf(common.DefaultClusterClearBashFile, cluster.Name)
		execClean        = fmt.Sprintf("chmod +x %[1]s && /bin/bash -c %[1]s", cleanFile)
		rmRootfs         = fmt.Sprintf("rm -rf %s", clusterRootfsDir)
		rmDockerCert     = fmt.Sprintf("rm -rf %s/%s*", runtime.DockerCertDir, runtime.SeaHub)
		envProcessor     = env.NewEnvProcessor(cluster)
	)

	eg, _ := errgroup.WithContext(context.Background())
	for _, IP := range ipList {
		ip := IP
		eg.Go(func() error {
			SSH, err := ssh.GetHostSSHClient(ip, cluster)
			if err != nil {
				return err
			}
			cmd := fmt.Sprintf("%s && %s", rmRootfs, rmDockerCert)
			if mounted, _ := mount.GetRemoteMountDetails(SSH, ip, clusterRootfsDir); mounted {
				cmd = fmt.Sprintf("umount %s && %s", clusterRootfsDir, cmd)
			}
			if exists, err := SSH.IsFileExist(ip, fmt.Sprintf(common.DefaultClusterClearBashFile, cluster.Name)); err != nil {
				return err
			} else if exists {
				cmd = fmt.Sprintf("%s && %s", execClean, cmd)
			}
			if err := SSH.CmdAsync(ip, envProcessor.WrapperShell(ip, cmd)); err != nil {
				return err
			}
			return nil
		})
	}
	return eg.Wait()
}

func NewOverlayFileSystem() (Interface, error) {
	return &overlayFileSystem{}, nil
}
