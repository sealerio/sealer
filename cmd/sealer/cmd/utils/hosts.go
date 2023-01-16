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

package utils

import (
	"net"

	v1 "github.com/sealerio/sealer/types/api/v1"

	"github.com/sealerio/sealer/common"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

// TransferIPToHosts constructs v2.Host through []net.ip
func TransferIPToHosts(masterIPList, nodeIPList []net.IP, sshAuthOnHosts v1.SSH) []v2.Host {
	var hosts []v2.Host
	if len(masterIPList) != 0 {
		hosts = append(hosts, v2.Host{
			Roles: []string{common.MASTER},
			IPS:   masterIPList,
			SSH:   sshAuthOnHosts,
		})
	}

	if len(nodeIPList) != 0 {
		hosts = append(hosts, v2.Host{
			Roles: []string{common.NODE},
			IPS:   nodeIPList,
			SSH:   sshAuthOnHosts,
		})
	}

	return hosts
}
