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

package apply

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/sealerio/sealer/apply/applydriver"
	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utilsnet "github.com/sealerio/sealer/utils/net"
)

func ConstructClusterFromArg(imageName string, runArgs *Args) (*v2.Cluster, error) {
	resultHosts, err := getHosts(runArgs.Masters, runArgs.Nodes)
	if err != nil {
		return nil, err
	}

	cluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			SSH: v1.SSH{
				User:     runArgs.User,
				Passwd:   runArgs.Password,
				PkPasswd: runArgs.PkPassword,
				Pk:       runArgs.Pk,
				Port:     strconv.Itoa(int(runArgs.Port)),
			},
			Image:   imageName,
			Hosts:   resultHosts,
			Env:     runArgs.CustomEnv,
			CMDArgs: runArgs.CMDArgs,
		},
	}
	cluster.APIVersion = common.APIVersion
	cluster.Kind = common.Kind
	cluster.Name = runArgs.ClusterName

	return &cluster, nil
}

func NewApplierFromArgs(imageName string, runArgs *Args) (applydriver.Interface, error) {
	if err := validateArgs(runArgs); err != nil {
		return nil, fmt.Errorf("failed to validate input run args: %v", err)
	}
	cluster, err := ConstructClusterFromArg(imageName, runArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cluster instance with command args: %v", err)
	}
	return NewDefaultApplier(cluster)
}

// validateArgs validates all the input args from sealer run command.
func validateArgs(runArgs *Args) error {
	// TODO: add detailed validation steps.
	var errMsg []string

	// validate input masters IP info
	if err := validateIPStr(runArgs.Masters); err != nil {
		errMsg = append(errMsg, err.Error())
	}

	// validate input nodes IP info
	if len(runArgs.Nodes) != 0 {
		// empty runArgs.Nodes are valid, since no nodes are input.
		if err := validateIPStr(runArgs.Nodes); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	if len(errMsg) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errMsg, ","))
}

// getHosts now only supports input IP list and IP range.
// IP list, like 192.168.0.1,192.168.0.2,192.168.0.3
// IP range, like 192.168.0.5-192.168.0.7, which means 192.168.0.5,192.168.0.6,192.168.0.7
// P.S. we have guaranteed that all the input masters and nodes are validated.
func getHosts(inMasters, inNodes string) ([]v2.Host, error) {
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
	masterHosts := make([]v2.Host, len(masters))
	for index, master := range masters {
		if index == 0 {
			// only master0 should add two roles: master and master0
			masterHosts = append(masterHosts, v2.Host{
				IPS: []net.IP{
					net.ParseIP(master),
				},
				Roles: []string{common.MASTER, common.MASTER0},
			})

			continue
		}

		masterHosts = append(masterHosts, v2.Host{
			IPS: []net.IP{
				net.ParseIP(master),
			},
			Roles: []string{common.MASTER},
		})
	}

	// if inNodes is empty,Split will return a slice of length 1 whose only element is inNodes.
	// so we need to filter the empty string to make sure the cluster node ip is valid.
	nodes := strings.Split(inNodes, ",")
	nodeHosts := make([]v2.Host, len(nodes))
	for _, node := range nodes {
		if node != "" {
			nodeHosts = append(nodeHosts, v2.Host{
				IPS:   []net.IP{net.ParseIP(node)},
				Roles: []string{common.NODE},
			})
		}
	}

	result := make([]v2.Host, len(masters)+len(nodes))
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
