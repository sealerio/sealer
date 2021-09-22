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
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/image/utils"

	"github.com/spf13/cobra"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:     "push",
	Short:   "push cloud image to registry",
	Example: `sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/my-kuberentes-cluster-with-dashboard:latest`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		imgsvc, err := image.NewImageService()
		if err != nil {
			return err
		}

		return imgsvc.Push(args[0])

	},
	ValidArgsFunction: utils.ImageListFuncForCompletion,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
