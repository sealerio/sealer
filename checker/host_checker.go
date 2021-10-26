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

	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"

	"github.com/alibaba/sealer/utils/ssh"
)

type HostChecker struct {
}

func (a HostChecker) Check(cluster *v1.Cluster, phase string) error {
	ssh, err := ssh.NewSSHClientWithCluster(cluster)
	if err != nil {
		return err
	}
	ipList := append(cluster.Spec.Masters.IPList, cluster.Spec.Nodes.IPList...)

	if HasDuplicateHostname(ssh.SSH, ipList) {
		return fmt.Errorf("hostname cannot be repeated, please set diffent hostname")
	}
	return TimeSync(ssh.SSH, ipList)
}

func NewHostChecker() Interface {
	return &HostChecker{}
}

func HasDuplicateHostname(s ssh.Interface, ipList []string) bool {
	hostnameList := map[string]bool{}
	for _, ip := range ipList {
		hostname, err := s.CmdToString(ip, "hostname", "")
		if err != nil {
			logger.Warn("failed to get host %s hostname, %v", ip, err)
		}
		if hostnameList[hostname] {
			return true
		}
		hostnameList[hostname] = true
	}
	return false
}

func TimeSync(s ssh.Interface, ipList []string) error {
	for _, ip := range ipList {
		timeStamp, err := s.CmdToString(ip, "date +%s", "")
		if err != nil {
			return fmt.Errorf("failed to get %s timestamp, %v", ip, err)
		}
		ts, err := strconv.Atoi(timeStamp)
		if err != nil {
			return fmt.Errorf("failed to reverse timestamp %s, %v", timeStamp, err)
		}
		timeDiff := time.Since(time.Unix(int64(ts), 0)).Minutes()
		if timeDiff < -1 || timeDiff > 1 {
			return fmt.Errorf("the time of %s node is not synchronized", ip)
		}
	}
	return nil
}
