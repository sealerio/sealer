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

	"github.com/sealerio/sealer/apply"

	netutils "github.com/sealerio/sealer/utils/net"
)

// ValidateRunArgs validates all the input args from sealer run command.
func ValidateRunArgs(runArgs *apply.Args) error {
	// TODO: add detailed validation steps.
	var errMsg []string

	// validate input masters IP info
	if err := ValidateIPStr(runArgs.Masters); err != nil {
		errMsg = append(errMsg, err.Error())
	}

	// validate input nodes IP info
	if len(runArgs.Nodes) != 0 {
		// empty runArgs.Nodes are valid, since no nodes are input.
		if err := ValidateIPStr(runArgs.Nodes); err != nil {
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

// ValidateJoinArgs validates all the input args from sealer join command.
func ValidateJoinArgs(joinArgs *apply.Args) error {
	var errMsg []string

	if joinArgs.Nodes == "" && joinArgs.Masters == "" {
		return fmt.Errorf("master and node cannot both be empty")
	}

	// validate input masters IP info
	if len(joinArgs.Masters) != 0 {
		if err := ValidateIPStr(joinArgs.Masters); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	// validate input nodes IP info
	if len(joinArgs.Nodes) != 0 {
		if err := ValidateIPStr(joinArgs.Nodes); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	if len(errMsg) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errMsg, ","))
}
