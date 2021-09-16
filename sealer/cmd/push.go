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
	"github.com/alibaba/sealer/pkg/image/utils"
	"github.com/alibaba/sealer/pkg/logger"

	"os"

	"github.com/spf13/cobra"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:     "push",
	Short:   "push cloud image to registry",
	Example: `sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/my-kuberentes-cluster-with-dashboard:latest`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		imgsvc, err := image.NewImageService()
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}

		if err = imgsvc.Push(args[0]); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
	ValidArgsFunction: utils.ImageListFuncForCompletion,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
