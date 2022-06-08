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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/utils/platform"
)

var platformFlag string

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "pull ClusterImage from a registry to local",
	// TODO: add long description.
	Long:    "",
	Example: `sealer pull registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		imgSvc, err := image.NewImageService()
		if err != nil {
			return err
		}

		plat, err := platform.GetPlatform(platformFlag)
		if err != nil {
			return err
		}
		if err := imgSvc.Pull(args[0], plat); err != nil {
			return err
		}
		logrus.Infof("Pull %s success", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().StringVar(&platformFlag, "platform", "", "set ClusterImage platform")
}
