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

package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
)

// if v1 >= v2 return true, else return false
func VersionCompare(v1, v2 string) bool {
	v1 = strings.Replace(v1, "v", "", -1)
	v2 = strings.Replace(v2, "v", "", -1)
	v1 = strings.Split(v1, "-")[0]
	v2 = strings.Split(v2, "-")[0]
	v1List := strings.Split(v1, ".")
	v2List := strings.Split(v2, ".")

	if len(v1List) != 3 || len(v2List) != 3 {
		logger.Error("error version format %s %s", v1, v2)
		return false
	}
	if v1List[0] > v2List[0] {
		return true
	} else if v1List[0] < v2List[0] {
		return false
	}
	if v1List[1] > v2List[1] {
		return true
	} else if v1List[1] < v2List[1] {
		return false
	}
	if v1List[2] > v2List[2] {
		return true
	}
	return true
}

func PreInitMaster0(sshClient ssh.Interface, remoteHostIP string) error {
	err := ssh.WaitSSHReady(sshClient, 6, remoteHostIP)
	if err != nil {
		return fmt.Errorf("apply cloud cluster failed: %s", err)
	}
	// send sealer and cluster file to remote host
	sealerPath := utils.ExecutableFilePath()
	err = sshClient.Copy(remoteHostIP, sealerPath, common.RemoteSealerPath)
	if err != nil {
		return fmt.Errorf("send sealer to remote host %s failed:%v", remoteHostIP, err)
	}
	err = sshClient.CmdAsync(remoteHostIP, fmt.Sprintf(common.ChmodCmd, common.RemoteSealerPath))
	if err != nil {
		return fmt.Errorf("chmod +x sealer on remote host %s failed:%v", remoteHostIP, err)
	}
	logger.Info("send sealer cmd to %s success !", remoteHostIP)

	// send tmp cluster file
	err = sshClient.Copy(remoteHostIP, common.TmpClusterfile, common.TmpClusterfile)
	if err != nil {
		return fmt.Errorf("send cluster file to remote host %s failed:%v", remoteHostIP, err)
	}
	logger.Info("send cluster file to %s success !", remoteHostIP)

	// send register login info
	authFile := common.DefaultRegistryAuthConfigDir()
	if utils.IsFileExist(authFile) {
		err = sshClient.Copy(remoteHostIP, authFile, common.DefaultRegistryAuthDir)
		if err != nil {
			return fmt.Errorf("failed to send register config %s to remote host %s err: %v", authFile, remoteHostIP, err)
		}
		logger.Info("send register info to %s success !", remoteHostIP)
	} else {
		logger.Warn("failed to find %s, if image registry is private, please login first", authFile)
	}
	return nil
}

func GetKubectlAndKubeconfig(ssh ssh.Interface, host string) error {
	// fetch the cluster kubeconfig, and add /etc/hosts "EIP apiserver.cluster.local" so we can get the current cluster status later
	err := ssh.Fetch(host, path.Join(common.DefaultKubeConfigDir(), "config"), common.KubeAdminConf)
	if err != nil {
		return errors.Wrap(err, "failed to copy kubeconfig")
	}
	err = utils.AppendFile(common.EtcHosts, fmt.Sprintf("%s %s", host, common.APIServerDomain))
	if err != nil {
		return errors.Wrap(err, "failed to append master IP to etc hosts")
	}
	err = ssh.Fetch(host, common.KubectlPath, common.KubectlPath)
	if err != nil {
		return errors.Wrap(err, "fetch kubectl failed")
	}
	err = utils.Cmd("chmod", "+x", common.KubectlPath)
	if err != nil {
		return errors.Wrap(err, "chmod a+x kubectl failed")
	}

	return nil
}

// LoadMetadata :read metadata via cluster image name.
func LoadMetadata(metadataPath string) (*Metadata, error) {
	var metadataFile []byte
	var err error
	var md Metadata
	if !utils.IsFileExist(metadataPath) {
		return nil, nil
	}

	metadataFile, err = ioutil.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CloudImage metadata %v", err)
	}
	err = json.Unmarshal(metadataFile, &md)
	if err != nil {
		return nil, fmt.Errorf("failed to load CloudImage metadata %v", err)
	}
	return &md, nil
}
