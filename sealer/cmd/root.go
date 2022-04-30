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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
)

type rootOpts struct {
	cfgFile     string
	debugModeOn bool
	hideLogTime bool
	hideLogPath bool
}

var rootOpt rootOpts

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sealer",
	Short: "",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&rootOpt.cfgFile, "config", "", "config file (default is $HOME/.sealer.json)")
	rootCmd.PersistentFlags().BoolVarP(&rootOpt.debugModeOn, "debug", "d", false, "turn on debug mode")
	rootCmd.PersistentFlags().BoolVar(&rootOpt.hideLogTime, "hide-time", false, "hide the log time")
	rootCmd.PersistentFlags().BoolVar(&rootOpt.hideLogPath, "hide-path", false, "hide the log path")
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

	logger.InitLogger(logger.Config{DebugMode: rootOpt.debugModeOn})

	logger.SetLogPath(!rootOpt.hideLogPath)

	if !rootOpt.hideLogTime {
		logger.SetTimeFormat(logger.LogTimeDefaultFormat)
	}
	logger.Cfg(rootOpt.debugModeOn)
}
