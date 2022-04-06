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
	"os"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image"
)

type LoginFlag struct {
	RegistryURL      string
	RegistryUsername string
	RegistryPasswd   string
}

var loginConfig *LoginFlag

var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "login image repository",
	Example: `sealer login registry.cn-qingdao.aliyuncs.com -u [username] -p [password]`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		imgSvc, err := image.NewImageService()
		if err != nil {
			return err
		}

		return imgSvc.Login(args[0], loginConfig.RegistryUsername, loginConfig.RegistryPasswd)
	},
}

func init() {
	loginConfig = &LoginFlag{}
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&loginConfig.RegistryUsername, "username", "u", "", "user name for login registry")
	loginCmd.Flags().StringVarP(&loginConfig.RegistryPasswd, "passwd", "p", "", "password for login registry")
	if err := loginCmd.MarkFlagRequired("username"); err != nil {
		logger.Error("failed to init flag: %v", err)
		os.Exit(1)
	}
	if err := loginCmd.MarkFlagRequired("passwd"); err != nil {
		logger.Error("failed to init flag: %v", err)
		os.Exit(1)
	}
}
