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
	"github.com/alibaba/sealer/pkg/exec"
	"github.com/spf13/cobra"
)

var roles string

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "exec a shell command or script on all node.",
	Example: `
exec to default cluster: my-cluster
	sealer exec "cat /etc/hosts"
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
    sealer exec -c my-cluster "cat /etc/hosts"
set role label to exec cmd:
    sealer exec -c my-cluster -r master,slave,node1 "cat /etc/hosts"		
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		execCmd, err := exec.NewExecCmd(clusterName, roles)
		if err != nil {
			return err
		}
		return execCmd.RunCmd(args[0])
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().StringVarP(&clusterName, "cluster-name", "c", "", "submit one cluster name")
	execCmd.Flags().StringVarP(&roles, "roles", "r", "", "set role label to roles")
}
