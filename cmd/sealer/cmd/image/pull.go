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
	"github.com/containers/buildah/pkg/parse"
	pkgauth "github.com/sealerio/sealer/pkg/auth"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/spf13/cobra"
)

var pullOpts *options.PullOptions

var exampleForPullCmd = `sealer pull registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
sealer pull registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 --platform linux/amd64
`

// NewPullCmd pullCmd represents the pull command
func NewPullCmd() *cobra.Command {
	pullCmd := &cobra.Command{
		Use:     "pull",
		Short:   "pull ClusterImage from a registry to local",
		Example: exampleForPullCmd,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}
			pullOpts.Image = args[0]
			return engine.Pull(pullOpts)
		},
	}
	pullOpts = &options.PullOptions{}
	pullCmd.Flags().StringVar(&pullOpts.Platform, "platform", parse.DefaultPlatform(), "prefer OS/ARCH instead of the current operating system and architecture for choosing images")
	pullCmd.Flags().StringVar(&pullOpts.Authfile, "authfile", pkgauth.GetDefaultAuthFilePath(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	pullCmd.Flags().BoolVar(&pullOpts.TLSVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	pullCmd.Flags().StringVar(&pullOpts.PullPolicy, "policy", "missing", "missing, always, or never.")
	pullCmd.Flags().BoolVarP(&pullOpts.Quiet, "quiet", "q", false, "don't output progress information when pulling images")
	return pullCmd
}
