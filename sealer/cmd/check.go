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
	"os"

	"github.com/alibaba/sealer/check/service"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/logger"
)

type CheckArgs struct {
	Pre  bool
	Post bool
}

var checkArgs *CheckArgs

// pushCmd represents the push command
var checkCmd = &cobra.Command{
	Use:     "check",
	Short:   "check the state of cluster ",
	Example: `sealer check --pre or sealer check --post`,
	Run: func(cmd *cobra.Command, args []string) {
		var checker service.CheckerService
		if checkArgs.Pre && checkArgs.Post {
			logger.Error("don't allow to set tow flags --pre and --post")
			os.Exit(1)
		} else if checkArgs.Pre {
			checker = service.NewPreCheckerService()
		} else if checkArgs.Post {
			checker = service.NewPostCheckerService()
		} else {
			checker = service.NewDefaultCheckerService()
		}
		if err := checker.Run(); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	checkArgs = &CheckArgs{}
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolVar(&checkArgs.Pre, "pre", false, "Check dependencies before cluster creation")
	checkCmd.Flags().BoolVar(&checkArgs.Post, "post", false, "Check the status of the cluster after it is created")
}
