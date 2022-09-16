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
	"strings"

	"github.com/sealerio/sealer/common"

	v2 "github.com/sealerio/sealer/types/api/v2"
	utilsnet "github.com/sealerio/sealer/utils/net"
)

// GetHosts now only supports input IP list and IP range.
// IP list, like 192.168.0.1,192.168.0.2,192.168.0.3
// IP range, like 192.168.0.5-192.168.0.7, which means 192.168.0.5,192.168.0.6,192.168.0.7
// P.S. we have guaranteed that all the input masters and nodes are validated.
func GetHosts(inMasters, inNodes string) ([]v2.Host, error) {
	var err error
	if isRange(inMasters) {
		inMasters, err = utilsnet.IPRangeToList(inMasters)
		if err != nil {
			return nil, err
		}
	}

	if isRange(inNodes) {
		if inNodes, err = utilsnet.IPRangeToList(inNodes); err != nil {
			return nil, err
		}
	}

	masters := strings.Split(inMasters, ",")
	masterHosts := make([]v2.Host, 0, len(masters))
	for _, master := range masters {
		if master == "" {
			continue
		}
		masterHosts = append(masterHosts, v2.Host{
			Roles: []string{common.MASTER},
			IPS:   []net.IP{net.ParseIP(master)},
		})
	}
	nodes := strings.Split(inNodes, ",")
	nodeHosts := make([]v2.Host, 0, len(nodes))
	for _, node := range nodes {
		if node == "" {
			continue
		}
		nodeHosts = append(nodeHosts, v2.Host{
			Roles: []string{common.NODE},
			IPS:   []net.IP{net.ParseIP(node)},
		})
	}
	result := make([]v2.Host, 0, len(masters)+len(nodes))
	result = append(result, masterHosts...)
	result = append(result, nodeHosts...)

	return result, nil
}

func isRange(ipStr string) bool {
	if len(ipStr) == 0 || !strings.Contains(ipStr, "-") {
		return false
	}
	return true
}
