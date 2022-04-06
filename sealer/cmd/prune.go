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
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/pkg/prune"

	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:     "prune",
	Short:   "prune sealer data dir",
	Args:    cobra.NoArgs,
	Example: `sealer prune`,
	RunE: func(cmd *cobra.Command, args []string) error {
		imService, err := image.NewImageService()
		if err != nil {
			return err
		}
		//TODO add more interfaces that need pruning
		for _, pruneService := range []prune.Interface{imService} {
			err := pruneService.Prune()
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}
