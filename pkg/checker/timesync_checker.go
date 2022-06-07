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

package checker

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sealerio/sealer/pkg/clusterinfo"
	v2 "github.com/sealerio/sealer/types/api/v2"
	sshutil "github.com/sealerio/sealer/utils/ssh"
)

type TimeSyncChecker struct {
}

func NewTimeSyncChecker() Interface {
	return &TimeSyncChecker{}
}

func (o TimeSyncChecker) Check(cluster *v2.Cluster, phase string) error {
	detailed, err := clusterinfo.GetClusterInfo(cluster)
	if err != nil {
		return err
	}
	var active = "active"
	var hasTimeSvcHosts []string
	var notHasTimeSvcHosts []string
	for _, instance := range detailed.InstanceInfos {
		host := instance.PrivateIP

		if instance.TimeSyncStatus.Ntpd == active && instance.TimeSyncStatus.Chronyd == active {
			return fmt.Errorf("host %s active ntpd.service and chronyd.service both, please disable one of them", host)
		}
		timeSvc := ""
		if instance.TimeSyncStatus.Ntpd == active {
			timeSvc = "ntp"
		}
		if instance.TimeSyncStatus.Chronyd == active {
			timeSvc = "chrony"
		}
		if timeSvc != "" {
			hasTimeSvcHosts = append(hasTimeSvcHosts, host)
			//Check time is sync
			s, err := sshutil.GetHostSSHClient(host, cluster)
			if err != nil {
				return fmt.Errorf("checker: failed to get host %s client,%v", host, err)
			}

			output, err := s.Cmd(host, "date +%s")
			if err != nil {
				return err
			}
			if output != nil {
				remoteTime, err := strconv.Atoi(strings.Replace(strings.Replace(string(output), "\r", "", -1), "\n", "", -1))
				if err != nil {
					return fmt.Errorf("get remote time of %s failed, error:%s", host, err.Error())
				}

				localTime := time.Now().Unix()
				timediff := int64(remoteTime) - localTime
				if (timediff > 5) || (-5 > timediff) {
					return fmt.Errorf("host %s has config %s, but its time diff between master0 greater than 5s", host, timeSvc)
				}
			} else { // command error
				return fmt.Errorf("get remote time of %s failed, output is nil", host)
			}
		} else {
			notHasTimeSvcHosts = append(notHasTimeSvcHosts, host)
		}
	}

	if len(hasTimeSvcHosts) == 0 {
		return fmt.Errorf("all hosts has no time sync service")
	} else if len(notHasTimeSvcHosts) == 0 {
		return nil
	}
	return fmt.Errorf("some hosts[%s] config time sync service, but some hosts[%s] not, please check",
		strings.Join(hasTimeSvcHosts, ","), strings.Join(notHasTimeSvcHosts, ","))
}
