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

package alpha

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/exec"
)

var (
	clusterName string
	roles       []string
)

var longExecCmdDescription = `Using ssh client which is built in sealer to run shell command on the nodes filtered by cluster and cluster roles. It is convenient for cluster administrator to do quick investigation.`

var exampleForExecCmd = `
Exec the default cluster node:
  sealer alpha exec "cat /etc/hosts"

specify the cluster name:
  sealer alpha exec -c my-cluster "cat /etc/hosts"

using role label to filter node and run exec cmd:
  sealer alpha exec -c my-cluster -r master,slave,node1 "cat /etc/hosts"		
`

// NewExecCmd implement the sealer exec command
func NewExecCmd() *cobra.Command {
	execCmd := &cobra.Command{
		Use:     "exec",
		Short:   "Exec a shell command or script on a specified node",
		Long:    longExecCmdDescription,
		Example: exampleForExecCmd,
		Args:    cobra.ExactArgs(1),
		RunE:    execActionFunc,
	}

	execCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "specify the name of cluster")
	execCmd.Flags().StringSliceVarP(&roles, "roles", "r", []string{}, "set role label to filter node")

	return execCmd
}

func execActionFunc(cmd *cobra.Command, args []string) error {
	var ipList []net.IP

	cluster, err := clusterfile.GetClusterFromFile(common.GetDefaultClusterfile())
	if err != nil {
		return err
	}

	if len(roles) == 0 {
		ipList = cluster.GetAllIPList()
	} else {
		for _, role := range roles {
			ipList = append(ipList, cluster.GetIPSByRole(role)...)
		}
		if len(ipList) == 0 {
			return fmt.Errorf("failed to get target ipList: no IP gotten by role(%s)", roles)
		}
	}

	execCmd := exec.NewExecCmd(cluster, ipList)
	return execCmd.RunCmd(args[0])
}
