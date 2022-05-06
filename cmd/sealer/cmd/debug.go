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
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/debug"
)

var debugOptions = debug.NewDebugOptions()

var debugCommand = &cobra.Command{
	Use:   "debug",
	Short: "Create debugging sessions for pods and nodes",
}

func init() {
	rootCmd.AddCommand(debugCommand)

	debugCommand.AddCommand(debug.CleanCMD)
	debugCommand.AddCommand(debug.ShowImagesCMD)
	debugCommand.AddCommand(debug.NewDebugPodCommand(debugOptions))
	debugCommand.AddCommand(debug.NewDebugNodeCommand(debugOptions))

	debugCommand.PersistentFlags().StringVar(&debugOptions.Image, "image", debugOptions.Image, "Container image to use for debug container.")
	debugCommand.PersistentFlags().StringVar(&debugOptions.DebugContainerName, "name", debugOptions.DebugContainerName, "Container name to use for debug container.")
	debugCommand.PersistentFlags().StringVar(&debugOptions.PullPolicy, "image-pull-policy", "IfNotPresent", "Container image pull policy, default policy is IfNotPresent.")
	debugCommand.PersistentFlags().StringSliceVar(&debugOptions.CheckList, "check-list", debugOptions.CheckList, "Check items, such as network、volume.")
	debugCommand.PersistentFlags().StringVarP(&debugOptions.Namespace, "namespace", "n", "default", "Namespace of Pod.")
	debugCommand.PersistentFlags().BoolVarP(&debugOptions.Interactive, "stdin", "i", debugOptions.Interactive, "Keep stdin open on the container, even if nothing is attached.")
	debugCommand.PersistentFlags().BoolVarP(&debugOptions.TTY, "tty", "t", debugOptions.TTY, "Allocate a TTY for the debugging container.")
	debugCommand.PersistentFlags().StringToStringP("env", "e", nil, "Environment variables to set in the container.")
}
