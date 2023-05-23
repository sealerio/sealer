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
	"github.com/spf13/cobra"
)

var longAlphaCmdDescription = `Alpha command of sealer is used to provide functionality incubation from immature to mature. Each function will experience a growing procedure. Alpha command policy calls on end users to experience alpha functionality as early as possible, and actively feedback the experience results to sealer community, and finally cooperate to promote function from incubation to graduation.

Please file an issue at https://github.com/sealerio/sealer/issues when you have any feedback on alpha commands.`

// NewCmdAlpha returns "sealer alpha" command.
func NewCmdAlpha() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alpha",
		Short: "sealer experimental sub-commands",
		Long:  longAlphaCmdDescription,
	}

	cmd.AddCommand(NewDebugCmd())
	cmd.AddCommand(NewExecCmd())
	cmd.AddCommand(NewMergeCmd())
	cmd.AddCommand(NewGenCmd())
	cmd.AddCommand(NewCheckCmd())
	cmd.AddCommand(NewSearchCmd())
	cmd.AddCommand(NewManifestCmd())
	cmd.AddCommand(NewHostAliasCmd())
	cmd.AddCommand(NewMountCmd())
	cmd.AddCommand(NewUmountCmd())
	return cmd
}
