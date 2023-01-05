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
	"strings"

	netutils "github.com/sealerio/sealer/utils/net"
)

// ValidateRunHosts validates the input host args such as master and node string
func ValidateRunHosts(runMasters, runNodes string) error {
	// TODO: add detailed validation steps.
	var errMsg []string

	// validate input masters IP info
	if len(runMasters) != 0 {
		if err := ValidateIPStr(runMasters); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	// validate input nodes IP info
	if len(runNodes) != 0 {
		// empty runFlags.Nodes are valid, since no nodes are input.
		if err := ValidateIPStr(runNodes); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	if len(errMsg) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errMsg, ","))
}

func ValidateIPStr(inputStr string) error {
	if len(inputStr) == 0 {
		return fmt.Errorf("input IP info cannot be empty")
	}

	// 1. validate if it is IP range
	if strings.Contains(inputStr, "-") {
		ips := strings.Split(inputStr, "-")
		if len(ips) != 2 {
			return fmt.Errorf("input IP(%s) is range format but invalid, IP range format must be xxx.xxx.xxx.1-xxx.xxx.xxx.70", inputStr)
		}

		if net.ParseIP(ips[0]) == nil {
			return fmt.Errorf("input IP(%s) is invalid", ips[0])
		}
		if net.ParseIP(ips[1]) == nil {
			return fmt.Errorf("input IP(%s) is invalid", ips[1])
		}

		if netutils.CompareIP(ips[0], ips[1]) >= 0 {
			return fmt.Errorf("input IP(%s) must be less than input IP(%s)", ips[0], ips[1])
		}

		return nil
	}

	// 2. validate if it is IP list, like 192.168.0.5,192.168.0.6,192.168.0.7
	for _, ip := range strings.Split(inputStr, ",") {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("input IP(%s) is invalid", ip)
		}
	}

	return nil
}

// ValidateScaleIPStr validates all the input args from scale up Or scale down command.
func ValidateScaleIPStr(masters, nodes string) error {
	var errMsg []string

	if nodes == "" && masters == "" {
		return fmt.Errorf("master and node cannot both be empty")
	}

	// validate input masters IP info
	if len(masters) != 0 {
		if err := ValidateIPStr(masters); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	// validate input nodes IP info
	if len(nodes) != 0 {
		if err := ValidateIPStr(nodes); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	if len(errMsg) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errMsg, ","))
}

// ParseToNetIPList now only supports input IP list and IP range.
// IP list, like 192.168.0.1,192.168.0.2,192.168.0.3
// IP range, like 192.168.0.5-192.168.0.7, which means 192.168.0.5,192.168.0.6,192.168.0.7
// P.S. we have guaranteed that all the input masters and nodes are validated.
func ParseToNetIPList(masters, workers string) ([]net.IP, []net.IP, error) {
	newMasters, err := netutils.TransferToIPList(masters)
	if err != nil {
		return nil, nil, err
	}

	newNodes, err := netutils.TransferToIPList(workers)
	if err != nil {
		return nil, nil, err
	}

	newMasterIPList := netutils.IPStrsToIPs(strings.Split(newMasters, ","))
	newNodeIPList := netutils.IPStrsToIPs(strings.Split(newNodes, ","))

	return newMasterIPList, newNodeIPList, nil
}
