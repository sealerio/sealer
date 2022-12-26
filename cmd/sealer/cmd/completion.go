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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/common"
)

var longCompletionCmdDescription = `Generate the autocompletion script for sealer for the bash shell.
To load completions in your current shell session:

	source <(sealer completion bash)

To load completions for every new session, execute once:

- Linux :
	## If bash-completion is not installed on Linux, please install the 'bash-completion' package
		sealer completion bash > /etc/bash_completion.d/sealer
	`

// NewCompletionCmd completionCmd represents the completion command
func NewCompletionCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:                   "completion",
		Short:                 "generate autocompletion script for bash",
		Long:                  longCompletionCmdDescription,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				if err := cmd.Root().GenBashCompletion(common.StdOut); err != nil {
					logrus.Errorf("failed to use bash completion, %v", err)
					os.Exit(1)
				}
			}
		},
	}
	return completionCmd
}
