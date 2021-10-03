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
	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:     "pull",
	Short:   "pull cloud image to local",
	Example: `sealer pull registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		imgSvc, err := image.NewImageService()
		if err != nil {
			return err
		}

		if err := imgSvc.Pull(args[0]); err != nil {
			return err
		}
		logger.Info("Pull %s success", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
