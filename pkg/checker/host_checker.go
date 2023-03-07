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
	"net"
	"strconv"
	"time"

	v2 "github.com/sealerio/sealer/types/api/v2"

	"github.com/sealerio/sealer/utils/ssh"
)

type HostChecker struct {
}

func (a HostChecker) Check(cluster *v2.Cluster, phase string) error {
	var ipList []net.IP
	for _, hosts := range cluster.Spec.Hosts {
		ipList = append(ipList, hosts.IPS...)
	}

	if err := checkHostnameUnique(cluster, ipList); err != nil {
		return err
	}
	return checkTimeSync(cluster, ipList)
}

func checkHostnameUnique(cluster *v2.Cluster, ipList []net.IP) error {
	hostnameList := map[string]bool{}
	for _, ip := range ipList {
		s, err := ssh.GetHostSSHClient(ip, cluster)
		if err != nil {
			return fmt.Errorf("checker: failed to get ssh client of host(%s): %v", ip, err)
		}
		hostname, err := s.CmdToString(ip, nil, "hostname", "")
		if err != nil {
			return fmt.Errorf("checker: failed to get hostname of host(%s): %v", ip, err)
		}
		if hostnameList[hostname] {
			return fmt.Errorf("checker: hostname of host(%s) cannot be repeated, please set diffent hostname", ip)
		}
		hostnameList[hostname] = true
	}
	return nil
}

// Check whether the node time is synchronized
func checkTimeSync(cluster *v2.Cluster, ipList []net.IP) error {
	for _, ip := range ipList {
		s, err := ssh.GetHostSSHClient(ip, cluster)
		if err != nil {
			return fmt.Errorf("checker: failed to get ssh client of host(%s): %v", ip, err)
		}
		timeStamp, err := s.CmdToString(ip, nil, "date +%s", "")
		if err != nil {
			return fmt.Errorf("checker: failed to get timestamp of host(%s): %v", ip, err)
		}
		ts, err := strconv.Atoi(timeStamp)
		if err != nil {
			return fmt.Errorf("checker: failed to reverse timestamp %s of host(%s): %v", timeStamp, ip, err)
		}
		timeDiff := time.Since(time.Unix(int64(ts), 0)).Minutes()
		if timeDiff < -1 || timeDiff > 1 {
			return fmt.Errorf("checker: time of host(%s) is not synchronized", ip)
		}
	}
	return nil
}
