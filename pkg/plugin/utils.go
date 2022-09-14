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

package plugin

import (
	"fmt"
	"net"
	"strings"

	v1 "github.com/sealerio/sealer/types/api/v1"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
	utilsnet "github.com/sealerio/sealer/utils/net"
)

const (
	DelSymbol   = "-"
	EqualSymbol = "="
	ColonSymbol = ":"
	SplitSymbol = "|"
)

func GetIpsByOnField(on string, context Context, phase Phase) (ipList []net.IP, err error) {
	splits := strings.Split(on, SplitSymbol)
	for _, split := range splits {
		var ips []net.IP
		split = strings.TrimSpace(split)
		switch {
		case strings.Contains(split, EqualSymbol):
			if (phase != PhasePostInstall && phase != PhasePostJoin) && phase != PhasePreClean {
				return nil, fmt.Errorf("current phase is %s. When nodes is specified with a label, the plugin action must be PostInstall or PostJoin. ", phase)
			}
			client, err := k8s.Newk8sClient()
			if err != nil {
				return nil, fmt.Errorf("failed to get k8s client: %v", err)
			}
			ips, err = client.ListNodeIPByLabel(strings.TrimSpace(split))
			if err != nil {
				return nil, err
			}
		case split == common.MASTER || split == common.NODE:
			ips = context.Cluster.GetIPSByRole(split)
		case split == common.MASTER0:
			ips = context.Cluster.GetIPSByRole(common.MASTER)
			if len(ips) < 1 {
				return nil, fmt.Errorf("invalid on field: [%s]", on)
			}
			ips = ips[:1]
		default:
			ips = utilsnet.DisassembleIPList(split)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("node not found by on field [%s]", on)
		}
		ipList = append(ipList, ips...)
	}
	return ipList, nil
}

func isSamePluginSpec(p1, p2 v1.Plugin) bool {
	return p1.Spec.Type == p2.Spec.Type && p1.Spec.On == p2.Spec.On &&
		p1.Spec.Data == p2.Spec.Data && p1.Spec.Action == p2.Spec.Action
}
