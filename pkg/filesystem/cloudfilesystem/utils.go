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
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/sealerio/sealer/utils/os/fs"

	osi "github.com/sealerio/sealer/utils/os"

	"github.com/sealerio/sealer/common"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/ssh"
)

func copyFiles(sshEntry ssh.Interface, ip, src, target string) error {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to copy files %s", err)
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

func copyRegistry(regIP string, cluster *v2.Cluster, mountDir map[string]bool, target string) error {
	sshClient, err := ssh.GetHostSSHClient(regIP, cluster)
	if err != nil {
		return err
	}
	for dir := range mountDir {
		dir = filepath.Join(dir, common.RegistryDirName)
		if !osi.IsFileExist(dir) {
			return nil
		}
		err = sshClient.Copy(regIP, dir, filepath.Join(target, common.RegistryDirName))
		if err != nil {
			return err
		}
	}
	return nil
}

func CleanFilesystem(clusterName string) error {
	return fs.NewFilesystem().RemoveAll(common.GetClusterWorkDir(clusterName), common.DefaultClusterBaseDir(clusterName),
		common.DefaultKubeConfigDir(), common.KubectlPath)
}
