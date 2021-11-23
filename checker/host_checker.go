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
	"strconv"
	"time"

	v1 "github.com/alibaba/sealer/types/api/v1"
	v2 "github.com/alibaba/sealer/types/api/v2"

	"github.com/alibaba/sealer/utils/ssh"
)

type HostChecker struct {
}

func (a HostChecker) Check(cluster *v2.Cluster, phase string) error {
	ssh, err := ssh.NewSSHClient(cluster.Spec.SSH)
	if err != nil {
		return err
	}
	err = checkHostnameUnique(ssh, cluster.Spec.Hosts)
	if err != nil {
		return err
	}
	return checkTimeSync(ssh, cluster.Spec.Hosts)
}

func NewHostChecker() Interface {
	return &HostChecker{}
}

func checkHostnameUnique(globalSSH ssh.Interface, HostList []v2.Hosts) error {
	hostnameList := map[string]bool{}
	var localSSH ssh.Interface
	for _, hosts := range HostList {
		if hosts.SSH != (v1.SSH{}) {
			localSSH = ssh.NewSSHClient(hosts.SSH)
		} else {
			localSSH = globalSSH
		}
		for _, IP := range hosts.IPS {
			hostname, err := localSSH.CmdToString(IP, "hostname", "")
			if err != nil {
				return fmt.Errorf("failed to get host %s hostname, %v", IP, err)
			}
			if hostnameList[hostname] {
				return fmt.Errorf("hostname cannot be repeated, please set diffent hostname")
			}
			hostnameList[hostname] = true
		}
	}
	return nil
}

//Check whether the node time is synchronized
func checkTimeSync(globalSSH ssh.Interface, HostList []v2.Hosts) error {
	var localSSH ssh.Interface
	for _, hosts := range HostList {
		if hosts.SSH != (v1.SSH{}) {
			localSSH = ssh.NewSSHClient(hosts.SSH)
		} else {
			localSSH = globalSSH
		}
		for _, IP := range hosts.IPS {
			timeStamp, err := localSSH.CmdToString(IP, "date +%s", "")
			if err != nil {
				return fmt.Errorf("failed to get %s timestamp, %v", IP, err)
			}
			ts, err := strconv.Atoi(timeStamp)
			if err != nil {
				return fmt.Errorf("failed to reverse timestamp %s, %v", timeStamp, err)
			}
			timeDiff := time.Since(time.Unix(int64(ts), 0)).Minutes()
			if timeDiff < -1 || timeDiff > 1 {
				return fmt.Errorf("the time of %s node is not synchronized", IP)
			}
		}
	}
	return nil
}
