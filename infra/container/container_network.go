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
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

func (c *DockerProvider) DeleteNetworkResource(id string) error {
	return c.DockerClient.NetworkRemove(c.Ctx, id)
}

func (c *DockerProvider) PrepareNetworkResource() error {
	networks, err := c.DockerClient.NetworkList(c.Ctx, types.NetworkListOptions{})
	if err != nil {
		return err
	}
	var targetIDs []string

	for _, net := range networks {
		if net.Name == c.NetworkResource.DefaultName {
			if len(net.Containers) > 1 {
				return fmt.Errorf("duplicate bridge name with default %s", net.Name)
			}
			targetIDs = append(targetIDs, net.ID)
		}
	}

	if len(targetIDs) > 0 {
		// reuse sealer network
		c.NetworkResource.ID = targetIDs[0]
		for i := 1; i < len(targetIDs); i++ {
			err = c.DeleteNetworkResource(targetIDs[i])
			if err != nil {
				return err
			}
		}
		return nil
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
		defaultBridge, err := c.DockerClient.NetworkInspect(c.Ctx, defaultBridgeID, types.NetworkInspectOptions{})
		if err != nil {
			return err
		}
		mtu = defaultBridge.Options["com.docker.network.driver.mtu"]
	}

	// create sealer network
	resp, err := c.DockerClient.NetworkCreate(c.Ctx, DefaultNetworkName, types.NetworkCreate{
		Driver:     "bridge",
		EnableIPv6: true,
		Options: map[string]string{
			"com.docker.network.bridge.enable_ip_masquerade": "true",
			"com.docker.network.driver.mtu":                  mtu,
		},
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{Subnet: GenerateSubnetFromName(c.NetworkResource.DefaultName, 0)},
			},
		},
	})

	if err != nil {
		return err
	}
	//create network and set id
	c.NetworkResource.ID = resp.ID
	return nil
}

func (c *DockerProvider) GetNetworkResourceByID(id string) (*types.NetworkResource, error) {
	net, err := c.DockerClient.NetworkInspect(c.Ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		return nil, err
	}

	return &net, err
}
