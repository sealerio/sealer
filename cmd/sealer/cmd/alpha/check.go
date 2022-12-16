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

	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/checker"
)

type CheckArgs struct {
	Pre  bool
	Post bool
}

var checkArgs *CheckArgs

var longNewCheckCmdDescription = `check command is used to check status of the cluster, including node status
, service status and pod status.`

var exampleForCheckCmd = `
  sealer check --pre 
  sealer check --post
`

// NewCheckCmd pushCmd represents the push command
func NewCheckCmd() *cobra.Command {
	checkCmd := &cobra.Command{
		Use:     "check",
		Short:   "check the state of cluster",
		Long:    longNewCheckCmdDescription,
		Example: exampleForCheckCmd,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkArgs.Pre && checkArgs.Post {
				return fmt.Errorf("don't allow to set two flags --pre and --post")
			}
			list := []checker.Interface{checker.NewNodeChecker(), checker.NewSvcChecker(), checker.NewPodChecker()}
			if checkArgs.Pre {
				return checker.RunCheckList(list, nil, checker.PhasePre)
			}
			return checker.RunCheckList(list, nil, checker.PhasePost)
		},
	}
	checkArgs = &CheckArgs{}
	checkCmd.Flags().BoolVar(&checkArgs.Pre, "pre", false, "Check dependencies before cluster creation")
	checkCmd.Flags().BoolVar(&checkArgs.Post, "post", false, "Check the status of the cluster after it is created")
	return checkCmd
}
