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

import (
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/auth"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
)

var pushOpts *options.PushOptions

var longNewPushCmdDescription = ``

var exampleForPushCmd = `
  sealer push docker.io/sealerio/kubernetes:v1-22-15-sealerio-2
`

// NewPushCmd pushCmd represents the push command
func NewPushCmd() *cobra.Command {
	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "push sealer image to remote registry",
		// TODO: add long description.
		Long:    longNewPushCmdDescription,
		Example: exampleForPushCmd,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}
			pushOpts.Image = args[0]
			return engine.Push(pushOpts)
		},
	}
	pushOpts = &options.PushOptions{}
	pushCmd.Flags().StringVar(&pushOpts.Authfile, "authfile", auth.GetDefaultAuthFilePath(), "path to store auth file after login. Accessing registry with this auth.")
	pushCmd.Flags().BoolVar(&pushOpts.SkipTLSVerify, "skip-tls-verify", false, "default is requiring HTTPS and verify certificates when accessing the registry.")
	pushCmd.Flags().BoolVarP(&pushOpts.Quiet, "quiet", "q", false, "don't output progress information when pushing images")
	pushCmd.Flags().BoolVar(&pushOpts.All, "all", true, "also push the images in the list")
	return pushCmd
}
