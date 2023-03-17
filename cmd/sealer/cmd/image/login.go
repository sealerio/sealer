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
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
)

var loginConfig *options.LoginOptions

var longNewLoginCmdDescription = ``

var exampleForLoginCmd = `
  sealer login registry.cn-qingdao.aliyuncs.com -u [username] -p [password]
`

func NewLoginCmd() *cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "login image registry",
		// TODO: add long description.
		Long:    longNewLoginCmdDescription,
		Example: exampleForLoginCmd,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adaptor, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
			if err != nil {
				return err
			}
			loginConfig.Domain = args[0]

			return adaptor.Login(loginConfig)
		},
	}
	loginConfig = &options.LoginOptions{}
	loginCmd.Flags().StringVarP(&loginConfig.Username, "username", "u", "", "user name for login registry")
	loginCmd.Flags().StringVarP(&loginConfig.Password, "passwd", "p", "", "password for login registry")
	loginCmd.Flags().BoolVar(&loginConfig.SkipTLSVerify, "skip-tls-verify", false, "default is requiring require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	if err := loginCmd.MarkFlagRequired("username"); err != nil {
		logrus.Errorf("failed to init flag: %v", err)
		os.Exit(1)
	}
	if err := loginCmd.MarkFlagRequired("passwd"); err != nil {
		logrus.Errorf("failed to init flag: %v", err)
		os.Exit(1)
	}
	return loginCmd
}
