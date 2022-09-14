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

package hostpool

import (
	"fmt"
)

// HostPool is a host resource pool of sealer's cluster, including masters and nodes.
// While SEALER DEPLOYING NODE has no restrict relationship with masters nor nodes:
// 1. sealer deploying node could be a node which is no master nor node;
// 2. sealer deploying node could also be one of masters and nodes.
// Then deploying node is not included in HostPool.
type HostPool struct {
	// host is a map:
	// key has a type of string which is from net.Ip.String()
	hosts map[string]*Host
}

// New initializes a brand new HostPool instance.
func New(hostConfigs []*HostConfig) (*HostPool, error) {
	if len(hostConfigs) == 0 {
		return nil, fmt.Errorf("input HostConfigs cannot be empty")
	}
	var hostPool HostPool
	for _, hostConfig := range hostConfigs {
		if _, OK := hostPool.hosts[hostConfig.IP.String()]; OK {
			return nil, fmt.Errorf("there must not be duplicated host IP(%s) in cluster hosts", hostConfig.IP.String())
		}
		hostPool.hosts[hostConfig.IP.String()] = &Host{
			config: HostConfig{
				IP:        hostConfig.IP,
				Port:      hostConfig.Port,
				User:      hostConfig.User,
				Password:  hostConfig.Password,
				Encrypted: hostConfig.Encrypted,
			},
		}
	}
	return &hostPool, nil
}

// Initialize helps HostPool to setup all attributes for each host,
// like scpClient, sshClient and so on.
func (hp *HostPool) Initialize() error {
	for _, host := range hp.hosts {
		if err := host.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize host in HostPool: %v", err)
		}
	}
	return nil
}

// GetHost gets the detailed host connection instance via IP string as a key.
func (hp *HostPool) GetHost(ipStr string) (*Host, error) {
	if host, exist := hp.hosts[ipStr]; exist {
		return host, nil
	}
	return nil, fmt.Errorf("cannot get host connection in HostPool by key(%s)", ipStr)
}
