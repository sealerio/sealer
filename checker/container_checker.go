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

package checker

import (
	"context"
	"fmt"
	"strings"
	"sync"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"
	"github.com/docker/docker/client"
)

const (
	SealerVersionTag = "-sealer"
	DockerInfo       = "docker info"
)

type NativeDockerChecker struct {
	SealerContainer bool
}

func (c NativeDockerChecker) Check(cluster *v1.Cluster, phase string) error {
	if phase == PhasePre {
		ipList := append(cluster.Spec.Masters.IPList, cluster.Spec.Nodes.IPList...)
		sshClient := ssh.NewSSHByCluster(cluster)
		var wg sync.WaitGroup
		var message []string
		for _, ip := range ipList {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				data, err := sshClient.CmdToString(ip, DockerInfo, " ")
				if err != nil {
					message = append(message, fmt.Sprintf("ssh connect failed in this host :%s", ip))
					return
				}
				version := strings.Contains(data, "Server Version:")
				if !version {
					return
				}
				c.SealerContainer = strings.Contains(data, SealerVersionTag)
				if !c.SealerContainer {
					message = append(message, fmt.Sprintf("docker already installed in this host : %s", ip))
				}
			}(ip)
		}
		wg.Wait()
		if len(message) > 0 {
			return fmt.Errorf("docker check failed. %s, please uninstall docker", message)
		}
		return nil
	}

	if phase == PhaseLiteBuild {
		ctx := context.Background()
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return fmt.Errorf("failed to init container client %s", err)
		}
		v, err := cli.ServerVersion(ctx)
		if err != nil {
			return nil
		}
		c.SealerContainer = strings.HasSuffix(v.Version, SealerVersionTag)
		if !c.SealerContainer {
			return fmt.Errorf("docker already installed in this host. docker version is : %s", v.Version)
		}
	}
	return nil
}

func NewNativeDockerChecker() Interface {
	return &NativeDockerChecker{
		SealerContainer: false,
	}
}
