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
	"strings"

	"github.com/alibaba/sealer/common"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
	"golang.org/x/sync/errgroup"
)

type ExecCmd struct {
	cluster *v2.Cluster
	ipList  []string
}

func NewExecCmd(clusterName string, roles string) (ExecCmd, error) {
	if clusterName == "" {
		var err error
		clusterName, err = utils.GetDefaultClusterName()
		if err != nil {
			return ExecCmd{}, err
		}
	}
	clusterFile := common.GetClusterWorkClusterfile(clusterName)
	cluster, err := utils.GetClusterFromFile(clusterFile)
	if err != nil {
		return ExecCmd{}, err
	}
	var ipList []string
	if roles == "" {
		ipList = append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	} else {
		roles := strings.Split(roles, ",")
		for _, role := range roles {
			ipList = append(ipList, cluster.GetIPSByRole(role)...)
		}
		if len(ipList) == 0 {
			return ExecCmd{}, fmt.Errorf("failed to get ipList, please check your roles label")
		}
	}
	return ExecCmd{cluster: cluster, ipList: ipList}, nil
}

func (exec ExecCmd) RunCmd(args ...string) error {
	eg, _ := errgroup.WithContext(context.Background())
	ipList := exec.ipList
	for _, ip := range ipList {
		ip := ip
		eg.Go(func() error {
			sshClient, sshErr := ssh.GetHostSSHClient(ip, exec.cluster)
			if sshErr != nil {
				return sshErr
			}
			err := sshClient.CmdAsync(ip, args...)
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to sealer exec command, err: %v", err)
	}
	return nil
}
