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
	"strings"

	netutils "github.com/sealerio/sealer/utils/net"
)

func validateIPStr(inputStr string) error {
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
