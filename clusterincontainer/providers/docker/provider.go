package docker

import (
	"context"
	"os"

	dockertypes "github.com/docker/docker/api/types"

	"github.com/alibaba/sealer/logger"

	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"

	"github.com/alibaba/sealer/clusterincontainer/providers"
	"github.com/alibaba/sealer/clusterincontainer/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// it won't compile if dockerprovider doesn't implement provider interface
var _ providers.Provider = &dockerprovider{}

type dockerprovider struct {
	client   *client.Client
	context  context.Context
	clusters []types.Cluster
}

func NewDockerProvider() (providers.Provider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Errorf("failed to create docker client, err: %v", err)
	}

	return &dockerprovider{
		client:  cli,
		context: context.Background(),
	}, nil
}

// Provision creates a cluster according to the cluster parameter
func (dp *dockerprovider) Provision(cluster types.Cluster) error {
	if len(cluster.Nodes) == 0 {
		return errors.Errorf("failed to create cluster, err: empty nodes")
	}

	for _, node := range cluster.Nodes {
		containerConfig := container.Config{
			Image:  node.Image,
			Labels: node.Labels,
		}
		hostconfig := container.HostConfig{
			Privileged: true,
		}

		for _, extraMount := range node.ExtraMounts {
			hostconfig.Mounts = append(hostconfig.Mounts, mount.Mount{
				Type:     mount.TypeBind,
				ReadOnly: extraMount.Readonly,
				Source:   extraMount.HostPath,
				Target:   extraMount.ContainerPath,
			})
		}

		_, err := dp.client.ContainerCreate(dp.context, &containerConfig, &hostconfig, nil, nil, node.Name)
		if err != nil {
			logger.Error("failed to create node %s, err: %v", node.Name, err)
			return err
		}
	}
	dp.clusters = append(dp.clusters, cluster)

	return nil
}

func (dp *dockerprovider) ListClusters() ([]string, error) {
	if len(dp.clusters) == 0 {
		return nil, errors.Errorf("failed to list clusters for there is no cluster under this provider")
	}

	var clusters []string
	for _, cluster := range dp.clusters {
		clusters = append(clusters, cluster.Name)
	}

	return clusters, nil
}

func (dp *dockerprovider) ListNodes(clusterName string) ([]types.Node, error) {
	if len(dp.clusters) == 0 {
		return nil, errors.Errorf("failed to list nodes for there is no cluster under this provider")
	}

	var nodes []types.Node
	for _, cluster := range dp.clusters {
		if cluster.Name != clusterName {
			continue
		}
		for _, node := range cluster.Nodes {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		return nil, errors.Errorf("failed to list nodes for there is no node in cluster "+
			"%s under this provider", clusterName)
	}

	return nodes, nil
}

func (dp *dockerprovider) DeleteNodes(nodes []types.Node) error {
	var (
		removeNodeMap map[string]bool
		options       dockertypes.ContainerRemoveOptions
		retErr        error
	)
	if len(nodes) == 0 {
		return nil
	}

	removeNodeMap = make(map[string]bool)
	options = dockertypes.ContainerRemoveOptions{
		Force: true,
	}

	for _, node := range nodes {
		err := dp.client.ContainerRemove(dp.context, node.Name, options)
		if err != nil {
			logger.Error("failed to remove node %s, err: %v", node.Name, err)
			retErr = err
			continue
		}
		removeNodeMap[node.Name] = true
	}
	var clusters []types.Cluster
	for _, cluster := range dp.clusters {
		var tempNodes []types.Node
		for _, node := range cluster.Nodes {
			if removeNodeMap[node.Name] {
				continue
			}
			tempNodes = append(tempNodes, node)
		}
		if tempNodes == nil {
			// cluster is empty, remove it
			continue
		}
		cluster.Nodes = tempNodes
		clusters = append(clusters, cluster)
	}

	dp.clusters = clusters

	return retErr
}

func (dp *dockerprovider) GetAPIServerEndpoint(clusterName string) (string, error) {
	for _, cluster := range dp.clusters {
		if cluster.Name == clusterName {
			for _, node := range cluster.Nodes {
				if node.Role == types.MasterRole {
					// return one of the master ip
					containerInfo, err := dp.client.ContainerInspect(dp.context, node.Name)
					if err != nil {
						return "", errors.Errorf("failed to get container info, err: %v", err)
					}

					if containerInfo.NetworkSettings.IPAddress == "" {
						return "", errors.Errorf("failed to get container ip, ipv4 address is empty")
					}

					return containerInfo.NetworkSettings.IPAddress + ":6443", nil
				}
			}
		}
	}

	return "", errors.Errorf("failed to find cluster %s in the provider", clusterName)
}

func (dp *dockerprovider) CollectLogs(dir string, nodes []types.Node) error {
	var (
		err  error
		file *os.File
	)
	if len(nodes) == 0 {
		return errors.Errorf("failed to collect logs for empty collection of nodes")
	}

	_, err = os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, tmpnode := range nodes {
		n := node{
			name: tmpnode.Name,
		}
		file, err = os.Create(dir + n.name + ".log")
		if err != nil {
			logger.Error("failed to create file %s, err: %v", dir+n.name, err)
			continue
		}
		err = n.SerialLogs(file)
		if err != nil {
			logger.Error("failed to collect node %s's log, err: %v", n.name, err)
			continue
		}
	}

	return err
}
