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
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/image/utils"
)

var removeOpts *options.RemoveImageOptions

// rmiCmd represents the rmi command
var rmiCmd = &cobra.Command{
	Use:   "rmi",
	Short: "remove local images by name",
	// TODO: add long description.
	Long:    "",
	Example: `sealer rmi registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`,
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRemove(args)
	},
	ValidArgsFunction: utils.ImageListFuncForCompletion,
}

func runRemove(images []string) error {
	removeOpts.ImageNamesOrIDs = images
	engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return err
	}

	return engine.RemoveImage(removeOpts)
}

func init() {
	removeOpts = &options.RemoveImageOptions{}
	flags := rmiCmd.Flags()
	flags.BoolVarP(&removeOpts.All, "all", "a", false, "remove all images")
	flags.BoolVarP(&removeOpts.Prune, "prune", "p", false, "prune dangling images")
	flags.BoolVarP(&removeOpts.Force, "force", "f", false, "force removal of the image and any containers using the image")
	rootCmd.AddCommand(rmiCmd)
}
