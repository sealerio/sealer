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
	"strings"
	"sync"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/env"
	"github.com/sealerio/sealer/pkg/runtime"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"
	strUtils "github.com/sealerio/sealer/utils/strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	RemoteChmod = "cd %s  && chmod +x scripts/* && cd scripts && bash init.sh /var/lib/docker %s %s %s"
)

type overlayFileSystem struct {
	ApplyMode string
}

func (o *overlayFileSystem) MountRootfs(cluster *v2.Cluster, hosts []string, initFlag bool) error {
	logrus.Debugf("start to mount root fs with overlay file system")
	clusterRootfsDir := common.DefaultTheClusterRootfsDir(cluster.Name)
	//scp roofs to all Masters and Nodes,then do init.sh
	if err := mountRootfs(hosts, clusterRootfsDir, cluster, initFlag, o.ApplyMode); err != nil {
		return fmt.Errorf("mount rootfs failed %v", err)
	}
	logrus.Debugf("success to mount rootfs")
	return nil
}

func (o *overlayFileSystem) UnMountRootfs(cluster *v2.Cluster, hosts []string) error {
	//do clean.sh,then remove all Masters and Nodes roofs
	if err := unmountRootfs(hosts, cluster); err != nil {
		return err
	}
	return nil
}

func mountRootfs(ipList []string, target string, cluster *v2.Cluster, initFlag bool, applyMode string) error {
	clusterPlatform, err := ssh.GetClusterPlatform(cluster)
	if err != nil {
		return err
	}
	mountEntry := struct {
		*sync.RWMutex
		mountDirs map[string]bool
	}{&sync.RWMutex{}, make(map[string]bool)}
	config := runtime.GetRegistryConfig(platform.DefaultMountClusterImageDir(cluster.Name), cluster.GetMaster0IP())
	containerRuntime := "docker"
	if v := env.ConvertEnv(cluster.Spec.Env)[v2.EnvContainerRuntime]; v != nil {
		containerRuntime = v.(string)
	}
	eg, _ := errgroup.WithContext(context.Background())
	for _, IP := range ipList {
		ip := IP
		eg.Go(func() error {
			src := platform.GetMountClusterImagePlatformDir(cluster.Name, clusterPlatform[ip])
			initCmd := fmt.Sprintf(RemoteChmod, target, config.Domain, config.Port, containerRuntime)
			mountEntry.Lock()
			if !mountEntry.mountDirs[src] {
				mountEntry.mountDirs[src] = true
			}
			mountEntry.Unlock()
			if applyMode == common.ApplyModeLoadImage {
				return nil
			}
			sshClient, err := ssh.GetHostSSHClient(ip, cluster, false)
			if err != nil {
				return fmt.Errorf("get host ssh client failed %v", err)
			}
			err = copyFiles(sshClient, ip, src, target)
			if err != nil {
				return fmt.Errorf("copy rootfs failed %v", err)
			}
			if initFlag {
				_, err = sshClient.CmdAsync(ip, env.NewEnvProcessor(cluster).WrapperShell(ip, initCmd))
				if err != nil {
					return fmt.Errorf("exec init.sh failed %v", err)
				}
			}
			return err
		})
	}
	if err = eg.Wait(); err != nil {
		return err
	}
	// if config.ip is not in mountRootfs ipList, mean copy registry dir is not required, like scale up node
	if applyMode != common.ApplyModeLoadImage && strUtils.NotIn(config.IP, ipList) {
		return nil
	}
	logrus.Debugf("start to copy registry")
	if err := copyRegistry(config.IP, cluster, mountEntry.mountDirs, target); err != nil {
		return err
	}
	logrus.Debugf("success to copy registry")
	return nil
}

func unmountRootfs(ipList []string, cluster *v2.Cluster) error {
	var (
		clusterRootfsDir = common.DefaultTheClusterRootfsDir(cluster.Name)
		cleanFile        = fmt.Sprintf(common.DefaultClusterClearBashFile, cluster.Name)
		unmount          = fmt.Sprintf("(! mountpoint -q %[1]s || umount -lf %[1]s)", clusterRootfsDir)
		execClean        = fmt.Sprintf("if [ -f \"%[1]s\" ];then chmod +x %[1]s && /bin/bash -c %[1]s;fi", cleanFile)
		rmRootfs         = fmt.Sprintf("rm -rf %s", clusterRootfsDir)
		envProcessor     = env.NewEnvProcessor(cluster)
		cmd              = strings.Join([]string{execClean, unmount, rmRootfs}, " && ")
	)

	eg, _ := errgroup.WithContext(context.Background())
	for _, IP := range ipList {
		ip := IP
		eg.Go(func() error {
			SSH, err := ssh.GetHostSSHClient(ip, cluster, false)
			if err != nil {
				return err
			}

			_, err = SSH.CmdAsync(ip, envProcessor.WrapperShell(ip, cmd))
			return err
		})
	}
	return eg.Wait()
}

func NewOverlayFileSystem(applyMode string) (Interface, error) {
	return &overlayFileSystem{
		ApplyMode: applyMode,
	}, nil
}
