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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/alibaba/sealer/pkg/env"

	"github.com/alibaba/sealer/pkg/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/pkg/image/store"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
	"github.com/alibaba/sealer/utils/ssh"
)

const (
	RemoteChmod = "cd %s  && chmod +x scripts/* && cd scripts && bash init.sh"
)

type Interface interface {
	MountRootfs(cluster *v2.Cluster, hosts []string, initFlag bool) error
	UnMountRootfs(cluster *v2.Cluster, hosts []string) error
	MountImage(cluster *v2.Cluster) error
	UnMountImage(cluster *v2.Cluster) error
	Clean(cluster *v2.Cluster) error
}

type FileSystem struct {
	imageStore store.ImageStore
}

func (c *FileSystem) MountImage(cluster *v2.Cluster) error {
	return c.mountImage(cluster)
}

func (c *FileSystem) UnMountImage(cluster *v2.Cluster) error {
	return c.umountImage(cluster)
}

func (c *FileSystem) MountRootfs(cluster *v2.Cluster, hosts []string, initFlag bool) error {
	clusterRootfsDir := common.DefaultTheClusterRootfsDir(cluster.Name)
	//scp roofs to all Masters and Nodes,then do init.sh
	if err := mountRootfs(hosts, clusterRootfsDir, cluster, initFlag); err != nil {
		return fmt.Errorf("mount rootfs failed %v", err)
	}
	return nil
}

func (c *FileSystem) UnMountRootfs(cluster *v2.Cluster, hosts []string) error {
	//do clean.sh,then remove all Masters and Nodes roofs
	if err := unmountRootfs(hosts, cluster); err != nil {
		return err
	}
	return nil
}

func (c *FileSystem) Clean(cluster *v2.Cluster) error {
	return utils.CleanFiles(common.GetClusterWorkDir(cluster.Name), common.DefaultClusterBaseDir(cluster.Name), common.DefaultKubeConfigDir(), common.KubectlPath)
}

func (c *FileSystem) umountImage(cluster *v2.Cluster) error {
	mountDir := common.DefaultMountCloudImageDir(cluster.Name)
	if !utils.IsFileExist(mountDir) {
		return nil
	}
	if isMount, _ := mount.GetMountDetails(mountDir); isMount {
		err := utils.Retry(10, time.Second, func() error {
			return mount.NewMountDriver().Unmount(mountDir)
		})
		if err != nil {
			return fmt.Errorf("failed to unmount dir %s,err: %v", mountDir, err)
		}
	}
	return os.RemoveAll(mountDir)
}

func (c *FileSystem) mountImage(cluster *v2.Cluster) error {
	var (
		mountdir = common.DefaultMountCloudImageDir(cluster.Name)
		upperDir = filepath.Join(mountdir, "upper")
		driver   = mount.NewMountDriver()
		err      error
	)
	if isMount, _ := mount.GetMountDetails(mountdir); isMount {
		err = driver.Unmount(mountdir)
		if err != nil {
			return fmt.Errorf("%s already mount, and failed to umount %v", mountdir, err)
		}
	}
	if utils.IsFileExist(mountdir) {
		err = os.RemoveAll(mountdir)
		if err != nil {
			return fmt.Errorf("failed to clean %s, %v", mountdir, err)
		}
	}
	//get layers
	Image, err := c.imageStore.GetByName(cluster.Spec.Image)
	if err != nil {
		return err
	}
	layers, err := image.GetImageLayerDirs(Image)
	if err != nil {
		return fmt.Errorf("get layers failed: %v", err)
	}

	if err = os.MkdirAll(upperDir, 0744); err != nil {
		return fmt.Errorf("create upperdir failed, %s", err)
	}
	if err = driver.Mount(mountdir, upperDir, layers...); err != nil {
		return fmt.Errorf("mount files failed %v", err)
	}
	return nil
}

func mountRootfs(ipList []string, target string, cluster *v2.Cluster, initFlag bool) error {
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
			err = CopyFiles(sshClient, ip == config.IP, ip, src, target)
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

func CopyFiles(sshEntry ssh.Interface, isRegistry bool, ip, src, target string) error {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to copy files %s", err)
	}

	if isRegistry {
		return sshEntry.Copy(ip, src, target)
	}
	for _, f := range files {
		if f.Name() == common.RegistryDirName {
			continue
		}
		err = sshEntry.Copy(ip, filepath.Join(src, f.Name()), filepath.Join(target, f.Name()))
		if err != nil {
			return fmt.Errorf("failed to copy sub files %v", err)
		}
	}
	return nil
}

func unmountRootfs(ipList []string, cluster *v2.Cluster) error {
	clusterRootfsDir := common.DefaultTheClusterRootfsDir(cluster.Name)
	execClean := fmt.Sprintf("/bin/bash -c "+common.DefaultClusterClearBashFile, cluster.Name)
	rmRootfs := fmt.Sprintf("rm -rf %s", clusterRootfsDir)
	rmDockerCert := fmt.Sprintf("rm -rf %s/%s*", runtime.DockerCertDir, runtime.SeaHub)
	envProcessor := env.NewEnvProcessor(cluster)
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
			if exists := SSH.IsFileExist(ip, fmt.Sprintf(common.DefaultClusterClearBashFile, cluster.Name)); exists {
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

func NewFilesystem() (Interface, error) {
	dis, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return &NydusFileSystem{&FileSystem{imageStore: dis}}, nil
}
