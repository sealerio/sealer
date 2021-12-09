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

package docker

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

func (p *Provider) DeleteNetworkResource(id string) error {
	return p.DockerClient.NetworkRemove(p.Ctx, id)
}

func (p *Provider) PrepareNetworkResource(networkName string) (string, error) {
	networks, err := p.DockerClient.NetworkList(p.Ctx, types.NetworkListOptions{})
	if err != nil {
		return "", err
	}
	var targetIDs []string

	for _, net := range networks {
		if net.Name == networkName {
			if len(net.Containers) > 1 {
				return "", fmt.Errorf("duplicate bridge name with default %s", net.Name)
			}
			targetIDs = append(targetIDs, net.ID)
		}
	}

	if len(targetIDs) > 0 {
		// reuse sealer network
		for i := 1; i < len(targetIDs); i++ {
			err = p.DeleteNetworkResource(targetIDs[i])
			if err != nil {
				return "", err
			}
		}
		return targetIDs[0], nil
	}

	defaultBridgeID := ""
	mtu := "1500"
	//get default bridge network id by name
	for _, net := range networks {
		if net.Name == "bridge" {
			defaultBridgeID = net.ID
			break
		}
	}

	// get default network bridge config
	if defaultBridgeID != "" {
		defaultBridge, err := p.DockerClient.NetworkInspect(p.Ctx, defaultBridgeID, types.NetworkInspectOptions{})
		if err != nil {
			return "", err
		}
		mtu = defaultBridge.Options["com.docker.network.driver.mtu"]
	}

	// create sealer network
	resp, err := p.DockerClient.NetworkCreate(p.Ctx, networkName, types.NetworkCreate{
		Driver:     "bridge",
		EnableIPv6: true,
		Options: map[string]string{
			"com.docker.network.bridge.enable_ip_masquerade": "true",
			"com.docker.network.driver.mtu":                  mtu,
		},
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{Subnet: GenerateSubnetFromName(networkName, 0)},
			},
		},
	})

	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (p *Provider) GetNetworkResourceByID(id string) (*types.NetworkResource, error) {
	net, err := p.DockerClient.NetworkInspect(p.Ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		return nil, err
	}

	return &net, err
}
