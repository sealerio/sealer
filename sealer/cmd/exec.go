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

package cmd

import (
	"sync"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
	"github.com/spf13/cobra"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "exec a shell command or script on all node.",
	Example: `
exec to default cluster: my-cluster
	sealer exec 'cat /etc/hosts'
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
    sealer exec -c my-cluster 'cat /etc/hosts'
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if clusterName == "" {
			var err error
			clusterName, err = utils.GetDefaultClusterName()
			if err != nil {
				return err
			}
		}
		clusterFile := common.GetClusterWorkClusterfile(clusterName)
		cluster, err := utils.GetClusterFromFile(clusterFile)
		if err != nil {
			return err
		}
		ipList := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
		var wg sync.WaitGroup
		for _, ip := range ipList {
			sshCli, err := ssh.GetHostSSHClient(ip, cluster)
			if err != nil {
				return err
			}
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				err := sshCli.CmdAsync(ip, args...)
				if err != nil {
					logger.Error("sealer exec failed %v", err)
				}
			}(ip)
		}
		wg.Wait()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "submit one cluster name")
}
