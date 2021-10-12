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

package upgrade

import (
	"fmt"

	"github.com/alibaba/sealer/utils/ssh"
)

//此文件下是不同linux发行版都要用到的函数

const (
	pullApiserver  = `docker pull registry.aliyuncs.com/google_containers/kube-apiserver:%s`
	pullController = `docker pull registry.aliyuncs.com/google_containers/kube-controller-manager:%s`
	pullScheduler  = `docker pull registry.aliyuncs.com/google_containers/kube-scheduler:%s`
	pullProxy      = `docker pull registry.aliyuncs.com/google_containers/kube-proxy:%s`
	tagApiserver   = `docker tag registry.aliyuncs.com/google_containers/kube-apiserver:%s sea.hub:5000/library/kube-apiserver:%s`
	tagController  = `docker tag registry.aliyuncs.com/google_containers/kube-controller-manager:%s sea.hub:5000/library/kube-controller-manager:%s`
	tagScheduler   = `docker tag registry.aliyuncs.com/google_containers/kube-scheduler:%s sea.hub:5000/library/kube-scheduler:%s`
	tagProxy       = `docker tag registry.aliyuncs.com/google_containers/kube-proxy:%s sea.hub:5000/library/kube-proxy:%s`
)

func pullAndTagDockerImage(client *ssh.Client, IP, version string) {
	var err error
	var dockerPullCmds = []string{
		fmt.Sprintf(pullApiserver, version),
		fmt.Sprintf(pullController, version),
		fmt.Sprintf(pullScheduler, version),
		fmt.Sprintf(pullProxy, version),
	}

	var dockerTagCmds = []string{
		fmt.Sprintf(tagApiserver, version, version),
		fmt.Sprintf(tagController, version, version),
		fmt.Sprintf(tagScheduler, version, version),
		fmt.Sprintf(tagProxy, version, version),
	}
	err = client.SSH.CmdAsync(IP, dockerPullCmds...)
	if err != nil {
		return
	}
	err = client.SSH.CmdAsync(IP, dockerTagCmds...)
	if err != nil {
		return
	}
}
