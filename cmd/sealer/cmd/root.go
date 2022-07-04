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
	"fmt"
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/pkg/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/version"
)

type rootOpts struct {
	cfgFile              string
	debugModeOn          bool
	hideLogTime          bool
	hideLogPath          bool
	colorMode            string
	remoteLoggerURL      string
	remoteLoggerTaskName string
}

var rootOpt rootOpts

const (
	colorModeNever  = "never"
	colorModeAlways = "always"
)

var supportedColorModes = []string{
	colorModeNever,
	colorModeAlways,
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sealer",
	Short: "A tool to build, share and run any distributed applications.",
	Long: `sealer is a tool to seal application's all dependencies and Kubernetes
into ClusterImage by Kubefile, distribute this application anywhere via ClusterImage, 
and run it within any cluster with Clusterfile in one command.
`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("sealer-%s: %v", version.GetSingleVersion(), err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&rootOpt.cfgFile, "config", "", "config file of sealer tool (default is $HOME/.sealer.json)")
	rootCmd.PersistentFlags().BoolVarP(&rootOpt.debugModeOn, "debug", "d", false, "turn on debug mode")
	rootCmd.PersistentFlags().BoolVarP(&rootCmd.SilenceUsage, "quiet", "q", false, "silence the usage when fail")
	rootCmd.PersistentFlags().BoolVar(&rootOpt.hideLogTime, "hide-time", false, "hide the log time")
	rootCmd.PersistentFlags().BoolVar(&rootOpt.hideLogPath, "hide-path", false, "hide the log path")
	rootCmd.PersistentFlags().StringVar(&rootOpt.colorMode, "color", colorModeAlways, fmt.Sprintf("set the log color mode, the possible values can be %v", supportedColorModes))
	rootCmd.PersistentFlags().StringVar(&rootOpt.remoteLoggerURL, "remote-logger-url", "", "remote logger url, if not empty, will send log to this url")
	rootCmd.PersistentFlags().StringVar(&rootOpt.remoteLoggerTaskName, "task-name", "", "task name which will embedded in the remote logger header, only valid when --remote-logger-url is set")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.DisableAutoGenTag = true
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if rootOpt.cfgFile == "" {
		// Find home directory.
		rootOpt.cfgFile = filepath.Join(common.GetHomeDir(), ".sealer.json")
	}
	// Use config file from the flag.
	// if not set config file, Search config in home directory with name ".sealer.json" (without extension).
	//viper.AddConfigPath(home)
	viper.SetConfigFile(rootOpt.cfgFile)

	viper.AutomaticEnv() // read in environment variables that match

	logger.New(logger.LogOptions{
		Verbose:              rootOpt.debugModeOn,
		RemoteLoggerURL:      rootOpt.remoteLoggerURL,
		RemoteLoggerTaskName: rootOpt.remoteLoggerTaskName,
		DisableColor:         rootOpt.colorMode == colorModeNever,
	})
}
