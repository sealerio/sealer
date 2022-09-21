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
	"fmt"
	"net"

	"github.com/sealerio/sealer/common"

	v2 "github.com/sealerio/sealer/types/api/v2"
)

// TransferIPStrToHosts now only supports input IP list and IP range.
// IP list, like 192.168.0.1,192.168.0.2,192.168.0.3
// IP range, like 192.168.0.5-192.168.0.7, which means 192.168.0.5,192.168.0.6,192.168.0.7
// P.S. we have guaranteed that all the input masters and nodes are validated.
func TransferIPStrToHosts(inMasters, inNodes string) ([]v2.Host, error) {
	masterIPList, nodeIPList, err := ParseToNetIPList(inMasters, inNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ip string to net IP list: %v", err)
	}

	masterHosts := make([]v2.Host, 0, len(masterIPList))
	for _, master := range masterIPList {
		masterHosts = append(masterHosts, v2.Host{
			Roles: []string{common.MASTER},
			IPS:   []net.IP{master},
		})
	}

	nodeHosts := make([]v2.Host, 0, len(nodeIPList))
	for _, node := range nodeIPList {
		nodeHosts = append(nodeHosts, v2.Host{
			Roles: []string{common.NODE},
			IPS:   []net.IP{node},
		})
	}

	return append(masterHosts, nodeHosts...), nil
}
