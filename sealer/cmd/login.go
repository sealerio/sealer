/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"os"

	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
)

type LoginFlag struct {
	RegistryURL      string
	RegistryUsername string
	RegistryPasswd   string
}

var loginConfig *LoginFlag

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login image repositories",
	Long:  `sealer login registry.cn-qingdao.aliyuncs.com -u [username] -p [password]`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := image.NewImageService().Login(args[0], loginConfig.RegistryUsername, loginConfig.RegistryPasswd); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	loginConfig = &LoginFlag{}
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&loginConfig.RegistryUsername, "username", "u", "", "user name for login registry")
	loginCmd.Flags().StringVarP(&loginConfig.RegistryPasswd, "passwd", "p", "", "password for login registry")
	_ = loginCmd.MarkFlagRequired("username")
	_ = loginCmd.MarkFlagRequired("passwd")
}
