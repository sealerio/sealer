// Copyright © 2022 Alibaba Group Holding Ltu.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implieu.
// See the License for the specific language governing permissions and
// limitations under the License.

package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/filesystem/cloudimage"
	"github.com/alibaba/sealer/pkg/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/platform"
	"github.com/alibaba/sealer/utils/ssh"
	"golang.org/x/sync/errgroup"
)

type UninstallProcessor struct {
	cloudImageMounter cloudimage.Interface
}

func (u *UninstallProcessor) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	todoList = append(todoList,
		u.Uninstall,
		u.UnMountRootfs,
		u.UnMountImage,
	)
	return todoList, nil
}

func (u *UninstallProcessor) Uninstall(cluster *v2.Cluster) error {
	var (
		rootfs = common.DefaultTheClusterRootfsDir(cluster.Name)
		lf     = fmt.Sprintf("%s/scripts/uninstall.sh", platform.DefaultMountCloudImageDir(cluster.Name))
		rf     = "/tmp/uninstall.sh"
		ip     = cluster.GetMaster0IP()
		cmd    = fmt.Sprintf("cd %s && chmod +x %s && /bin/bash -c %s", rootfs, rf, rf)
	)

	if !utils.IsExist(lf) {
		return nil
	}

	sshClient, err := ssh.NewStdoutSSHClient(ip, cluster)
	if err != nil {
		return err
	}

	err = sshClient.Copy(ip, lf, rf)
	if err != nil {
		return err
	}

	return sshClient.CmdAsync(ip, cmd)
}

func (u *UninstallProcessor) UnMountRootfs(cluster *v2.Cluster) error {
	var (
		// local rootfs mounter dir
		dir1 = common.DefaultTheClusterRootfsDir(cluster.Name)
		// app mounter dir
		dir2 = platform.DefaultMountCloudImageDir(cluster.Name)
	)

	deletedFile, err := utils.Sub(dir1, dir2)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("rm -rf %s", strings.Join(deletedFile, " "))
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	config := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), runtime.GetMaster0Ip(cluster))
	if utils.NotIn(config.IP, hosts) {
		hosts = append(hosts, config.IP)
	}

	// do delete action
	eg, _ := errgroup.WithContext(context.Background())
	for _, IP := range hosts {
		ip := IP
		eg.Go(func() error {
			sshClient, err := ssh.GetHostSSHClient(ip, cluster)
			if err != nil {
				return err
			}
			return sshClient.CmdAsync(ip, cmd)
		})
	}
	return eg.Wait()
}

func (u *UninstallProcessor) UnMountImage(cluster *v2.Cluster) error {
	return u.cloudImageMounter.UnMountImage(cluster)
}

func NewUninstallProcessor(mounter cloudimage.Interface) (Processor, error) {
	return &UninstallProcessor{
		cloudImageMounter: mounter,
	}, nil
}
