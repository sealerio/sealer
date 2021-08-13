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

package checker

import (
	"fmt"

	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
)

type HostChecker struct {
	SSH    ssh.Interface
	ipList []string
}

func (a HostChecker) Check() error {
	var hostnameList []string
	for _, ip := range a.ipList {
		err := ssh.WaitSSHReady(a.SSH, ip)
		if err != nil {
			return err
		}
		//hostname check
		hostname, err := a.SSH.CmdToString(ip, "hostname", "")
		if err != nil {
			return fmt.Errorf("failed to get host %s hostname, %v", ip, err)
		}
		hostnameList = append(hostnameList, hostname)
		//TODO CPU, Memory or more Check
	}

	if len(hostnameList) != len(utils.RemoveDuplicate(hostnameList)) {
		return fmt.Errorf("hostname cannot be repeated, please set diffent hostname")
	}
	return nil
}

func NewHostChecker(s ssh.Interface, ipList []string) Checker {
	return &HostChecker{s, ipList}
}
