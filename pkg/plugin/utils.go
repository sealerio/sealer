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
	"strings"

	v1 "github.com/sealerio/sealer/types/api/v1"

	"github.com/sealerio/sealer/utils/net"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
)

const (
	DelSymbol   = "-"
	EqualSymbol = "="
	ColonSymbol = ":"
	SplitSymbol = "|"
)

func GetIpsByOnField(on string, context Context, phase Phase) (ipList []string, err error) {
	on = strings.TrimSpace(on)
	if strings.Contains(on, EqualSymbol) {
		if (phase != PhasePostInstall && phase != PhasePostJoin) && phase != PhasePreClean {
			logrus.Warnf("Current phase is %s. When nodes is specified with a label, the plugin action must be PostInstall or PostJoin, ", phase)
			return nil, nil
		}
		client, err := k8s.Newk8sClient()
		if err != nil {
			return nil, fmt.Errorf("failed to get k8s client: %v", err)
		}
		ipList, err = client.ListNodeIPByLabel(strings.TrimSpace(on))
		if err != nil {
			return nil, err
		}
	} else if on == common.MASTER || on == common.NODE {
		ipList = context.Cluster.GetIPSByRole(on)
	} else if on == common.MASTER0 {
		ipList = context.Cluster.GetIPSByRole(common.MASTER)
		if len(ipList) < 1 {
			return nil, fmt.Errorf("invalid on filed: [%s]", on)
		}
		ipList = ipList[:1]
	} else {
		ipList = net.DisassembleIPList(on)
	}
	if len(ipList) == 0 {
		logrus.Debugf("node not found by on field [%s]", on)
	}
	return ipList, nil
}

func isSamePluginSpec(p1, p2 v1.Plugin) bool {
	return p1.Spec.Type == p2.Spec.Type && p1.Spec.On == p2.Spec.On &&
		p1.Spec.Data == p2.Spec.Data && p1.Spec.Action == p2.Spec.Action
}
