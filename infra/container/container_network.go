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
	var targetIds []string

	for _, net := range networks {
		if net.Name == c.NetworkResource.DefaultName {
			if len(net.Containers) > 1 {
				return fmt.Errorf("duplicate bridge name with default %s", net.Name)
			}
			targetIds = append(targetIds, net.ID)
		}
	}

	if len(targetIds) > 0 {
		// reuse sealer network
		c.NetworkResource.Id = targetIds[0]
		for i := 1; i < len(targetIds); i++ {
			err = c.DeleteNetworkResource(targetIds[i])
			if err != nil {
				return err
			}
		}
		return nil
	}

	defaultBridgeId := ""
	mtu := "1500"
	//get default bridge network id by name
	for _, net := range networks {
		if net.Name == "bridge" {
			defaultBridgeId = net.ID
			break
		}
	}

	// get default network bridge config
	if defaultBridgeId != "" {
		defaultBridge, err := c.DockerClient.NetworkInspect(c.Ctx, defaultBridgeId, types.NetworkInspectOptions{})
		if err != nil {
			return err
		}
		mtu = defaultBridge.Options["com.docker.network.driver.mtu"]
	}

	// create sealer network
	resp, err := c.DockerClient.NetworkCreate(c.Ctx, DEFAULT_NETWORK_NAME, types.NetworkCreate{
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
	c.NetworkResource.Id = resp.ID
	return nil
}

func (c *DockerProvider) GetNetworkResourceById(id string) (*types.NetworkResource, error) {
	net, err := c.DockerClient.NetworkInspect(c.Ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		return nil, err
	}

	return &net, err
}
