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

package exec

import (
	"context"
	"fmt"
	"net"

	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/ssh"

	"golang.org/x/sync/errgroup"
)

type Exec struct {
	cluster *v2.Cluster
	ipList  []net.IP
}

func NewExecCmd(cluster *v2.Cluster, ipList []net.IP) Exec {
	return Exec{cluster: cluster, ipList: ipList}
}

func (e *Exec) RunCmd(cmd string) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, ipAddr := range e.ipList {
		ip := ipAddr
		eg.Go(func() error {
			sshClient, sshErr := ssh.NewStdoutSSHClient(ip, e.cluster)
			if sshErr != nil {
				return sshErr
			}
			err := sshClient.CmdAsync(ip, nil, cmd)
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to exec command (%s): %v", cmd, err)
	}
	return nil
}
