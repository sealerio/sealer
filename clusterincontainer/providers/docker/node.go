/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliep.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"context"
	"io"
	"strings"

	"github.com/docker/docker/client"

	"github.com/alibaba/sealer/clusterincontainer/nodes"

	"github.com/pkg/errors"

	"github.com/alibaba/sealer/clusterincontainer/types"
	dockertypes "github.com/docker/docker/api/types"
)

const (
	SEP = "-"
)

var _ nodes.Node = &node{}

type node struct {
	name    string
	role    string
	ipv4    string
	ipv6    string
	client  *client.Client
	context context.Context
}

func NewDockerNode() (nodes.Node, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Errorf("failed to create docker client, err: %v", err)
	}

	return &node{
		client:  cli,
		context: context.Background(),
	}, nil
}

func (n *node) String() string {
	return n.name
}

// Role returns the role of the node.
// Because a container running node image is a node,
// so we can judge a node's role by its name or by setting labels
// For simplicity, we choose the former.
// Master nodes: clustername-master-x
// Worker nodes: clustername-worker-x
func (n *node) Role() (string, error) {
	if n.role != "" && (n.role == string(types.MasterRole) ||
		n.role == string(types.WorkerRole)) {
		return n.role, nil
	}

	arr := strings.Split(n.name, SEP)
	if len(arr) < 3 || arr[0] != "sealer" || arr[1] == "" {
		return "", errors.Errorf("node name format incorret, wanted: e.g. sealer-master-1)")
	}

	switch arr[1] {
	case string(types.MasterRole):
		n.role = string(types.MasterRole)
	case string(types.WorkerRole):
		n.role = string(types.WorkerRole)
	default:
		return "", errors.Errorf("failed to get node's from  node name")
	}

	return n.role, nil
}

func (n *node) IP() (ipv4 string, ipv6 string, err error) {
	if n.ipv4 != "" && n.ipv6 != "" {
		return n.ipv4, n.ipv6, nil
	}

	container, err := n.client.ContainerInspect(n.context, n.name)
	if err != nil {
		return "", "", errors.Errorf("failed to get container info, err: %v", err)
	}

	if container.NetworkSettings.IPAddress == "" {
		return "", "", errors.Errorf("failed to get container ip, ipv4 address is empty")
	}

	ipv4 = container.NetworkSettings.IPAddress
	ipv6 = container.NetworkSettings.GlobalIPv6Address

	return ipv4, ipv6, nil
}

func (n *node) SerialLogs(w io.Writer) error {
	options := dockertypes.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
		Details:    true,
	}

	reader, err := n.client.ContainerLogs(n.context, n.name, options)
	if err != nil {
		return errors.Errorf("failed to get container %s's log, err: %v", n.name, err)
	}

	_, err = io.Copy(w, reader)
	if err != nil {
		return errors.Errorf("failed to get container %s's log, err: %v", n.name, err)
	}

	return nil
}

func (n *node) Command(command string, args ...string) types.Cmd {
	return &nodeCmd{
		nameOrID: n.name,
		command:  command,
		args:     args,
		client:   n.client,
	}
}

func (n *node) CommandContext(ctx context.Context, command string, args ...string) types.Cmd {
	return &nodeCmd{
		nameOrID: n.name,
		command:  command,
		args:     args,
		ctx:      ctx,
		client:   n.client,
	}
}

var _ types.Cmd = &nodeCmd{}

// nodeCmd implements exec.Cmd for docker nodes
type nodeCmd struct {
	nameOrID string // the container name or ID
	command  string
	args     []string
	env      []string
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	ctx      context.Context
	client   *client.Client
}

// Run executes a command inside docker containers
func (c *nodeCmd) Run() error {
	var args []string

	// specify the container and command, after this everything will be
	// args the command in the container rather than to docker
	args = append(
		args,
		c.command, // with the command specified
	)
	args = append(
		args,
		// finally, with the caller args
		c.args...,
	)

	config := dockertypes.ExecConfig{
		Privileged:   true,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          c.env,
		Cmd:          args,
	}
	_, err := c.client.ContainerExecCreate(c.ctx, c.nameOrID, config)
	return err
}

func (c *nodeCmd) SetEnv(env ...string) types.Cmd {
	c.env = env
	return c
}

func (c *nodeCmd) SetStdin(r io.Reader) types.Cmd {
	c.stdin = r
	return c
}

func (c *nodeCmd) SetStdout(w io.Writer) types.Cmd {
	c.stdout = w
	return c
}

func (c *nodeCmd) SetStderr(w io.Writer) types.Cmd {
	c.stderr = w
	return c
}
