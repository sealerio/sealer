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

package image

import "github.com/spf13/cobra"

func NewCmdImage() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.AddCommand(NewBuildCmd())
	cmd.AddCommand(NewGenDocCommand())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewInspectCmd())
	cmd.AddCommand(NewLoadCmd())
	cmd.AddCommand(NewLoginCmd())
	cmd.AddCommand(NewLogoutCmd())
	cmd.AddCommand(NewPullCmd())
	cmd.AddCommand(NewPushCmd())
	cmd.AddCommand(NewRmiCmd())
	cmd.AddCommand(NewSaveCmd())
	cmd.AddCommand(NewSearchCmd())
	cmd.AddCommand(NewTagCmd())
	return cmd
}
