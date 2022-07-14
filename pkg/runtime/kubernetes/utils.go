// Copyright © 2022 Alibaba Group Holding Ltd.
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

package kubernetes

import (
	"fmt"

	"net"
	"path"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/utils/exec"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/ssh"

	"github.com/pkg/errors"
)

//KubeVersion is a Metadata kubernetes version field such as "v1.19.8"
type KubeVersion struct {
	FieldList []string
}

// Version swap string to []string
func (v *KubeVersion) Version(v1 string) KubeVersion {
	v1 = strings.Replace(v1, "v", "", -1)
	v1 = strings.Split(v1, "-")[0]
	return KubeVersion{
		FieldList: strings.Split(v1, "."),
	}
}

// Compare :if v >= v1 return true, else return false
func (v *KubeVersion) Compare(v1 KubeVersion) (bool, error) {
	if len(v.FieldList) != 3 || len(v1.FieldList) != 3 {
		return false, fmt.Errorf("error version format %s %s", v, v1)
	}
	//TODO: check if necessary need v = v1 logic!
	if v.FieldList[0] > v1.FieldList[0] {
		return true, nil
	} else if v.FieldList[0] < v1.FieldList[0] {
		return false, nil
	}
	if v.FieldList[1] > v1.FieldList[1] {
		return true, nil
	} else if v.FieldList[1] < v1.FieldList[1] {
		return false, nil
	}
	if v.FieldList[2] > v1.FieldList[2] {
		return true, nil
	}
	return true, nil
}

func GetKubectlAndKubeconfig(ssh ssh.Interface, host net.IP, rootfs string) error {
	// fetch the cluster kubeconfig, and add /etc/hosts "EIP apiserver.cluster.local" so we can get the current cluster status later
	err := ssh.Fetch(host, path.Join(common.DefaultKubeConfigDir(), "config"), common.KubeAdminConf)
	if err != nil {
		return errors.Wrap(err, "failed to copy kubeconfig")
	}
	_, err = exec.RunSimpleCmd(fmt.Sprintf("cat /etc/hosts |grep '%s %s' || echo '%s %s' >> /etc/hosts",
		host, common.APIServerDomain, host, common.APIServerDomain))
	if err != nil {
		return errors.Wrap(err, "failed to add master IP to etc hosts")
	}

	if !osi.IsFileExist(common.KubectlPath) {
		err = osi.RecursionCopy(filepath.Join(rootfs, "bin/kubectl"), common.KubectlPath)
		if err != nil {
			return err
		}
		err = exec.Cmd("chmod", "+x", common.KubectlPath)
		if err != nil {
			return errors.Wrap(err, "failed to chmod a+x kubectl")
		}
	}
	return nil
}
