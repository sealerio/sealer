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

package clusterruntime

import (
	"net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/infradriver"
)

func getWorkerIPList(infraDriver infradriver.InfraDriver) []net.IP {
	masters := make(map[string]bool)
	for _, master := range infraDriver.GetHostIPListByRole(common.MASTER) {
		masters[master.String()] = true
	}
	all := infraDriver.GetHostIPList()
	workers := make([]net.IP, len(all)-len(masters))

	index := 0
	for _, ip := range all {
		if !masters[ip.String()] {
			workers[index] = ip
			index++
		}
	}

	return workers
}
