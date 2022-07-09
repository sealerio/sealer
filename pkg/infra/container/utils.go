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

package container

import (
	"crypto/rand"
	"fmt"
	"net"
	"strconv"
	"strings"

	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/exec"
)

func IsDockerAvailable() bool {
	lines, err := exec.RunSimpleCmd("docker -v")
	if err != nil || len(lines) != 1 {
		return false
	}
	return strings.Contains(lines, "docker version")
}

func getDiff(host v1.Hosts) (int, []net.IP, error) {
	var num int
	var iplist []net.IP
	count, err := strconv.Atoi(host.Count)
	if err != nil {
		return 0, nil, err
	}
	if count > len(host.IPList) {
		//scale up
		num = count - len(host.IPList)
	}

	if count < len(host.IPList) {
		//scale down
		iplist = host.IPList[count:]
	}

	return num, iplist, nil
}

func GenUniqueID(n int) string {
	randBytes := make([]byte, n/2)
	if _, err := rand.Read(randBytes); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", randBytes)
}
