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

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/client/k8s"
	"github.com/alibaba/sealer/utils"
)

const (
	DelSymbol   = "-"
	EqualSymbol = "="
	ColonSymbol = ":"
)

func GetIpsByOnField(on string, context Context, phase Phase) (ipList []string, err error) {
	on = strings.TrimSpace(on)
	if strings.Contains(on, EqualSymbol) {
		if phase != PhasePostInstall {
			return nil, fmt.Errorf("the action must be PostInstall, When nodes is specified with a label")
		}
		client, err := k8s.Newk8sClient()
		if err != nil {
			return nil, err
		}
		ipList, err = client.ListNodeIPByLabel(strings.TrimSpace(on))
		if err != nil {
			return nil, err
		}
	} else if on == common.MASTER || on == common.NODE {
		ipList = context.Cluster.GetIPSByRole(on)
	} else {
		ipList = utils.DisassembleIPList(on)
	}
	if len(ipList) == 0 {
		return nil, fmt.Errorf("invalid on filed: [%s]", on)
	}
	return ipList, nil
}
