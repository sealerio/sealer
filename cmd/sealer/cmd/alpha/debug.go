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

package alpha

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/debug"
)

// NewDebugCmd returns the sealer debug Cobra command
func NewDebugCmd() *cobra.Command {
	var debugOptions = debug.NewDebugOptions()

	var debugCommand = &cobra.Command{
		Use:   "debug",
		Short: "Create debugging sessions for pods and nodes",
		// TODO: add long description.
		Long: "",
	}

	debugCommand.AddCommand(newDebugCleanCMD())
	debugCommand.AddCommand(newDebugShowImageCMD())
	debugCommand.AddCommand(newDebugPodCommand(debugOptions))
	debugCommand.AddCommand(newDebugNodeCommand(debugOptions))

	debugCommand.PersistentFlags().StringVar(&debugOptions.Image, "image", debugOptions.Image, "Container image to use for debug container.")
	debugCommand.PersistentFlags().StringVar(&debugOptions.DebugContainerName, "name", debugOptions.DebugContainerName, "Container name to use for debug container.")
	debugCommand.PersistentFlags().StringVar(&debugOptions.PullPolicy, "image-pull-policy", "IfNotPresent", "Container image pull policy, default policy is IfNotPresent.")
	debugCommand.PersistentFlags().StringSliceVar(&debugOptions.CheckList, "check-list", debugOptions.CheckList, "Check items, such as network, volume.")
	debugCommand.PersistentFlags().StringVarP(&debugOptions.Namespace, "namespace", "n", "default", "Namespace of Pod.")
	debugCommand.PersistentFlags().BoolVarP(&debugOptions.Interactive, "stdin", "i", debugOptions.Interactive, "Keep stdin open on the container, even if nothing is attached.")
	debugCommand.PersistentFlags().BoolVarP(&debugOptions.TTY, "tty", "t", debugOptions.TTY, "Allocate a TTY for the debugging container.")
	debugCommand.PersistentFlags().StringToStringP("env", "e", nil, "Environment variables to set in the container.")

	return debugCommand
}

func newDebugCleanCMD() *cobra.Command {
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean the debug container od pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cleaner := debug.NewDebugCleaner()
			cleaner.AdminKubeConfigPath = common.KubeAdminConf

			if err := cleaner.CompleteAndVerifyOptions(args); err != nil {
				return err
			}
			if err := cleaner.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cleanCmd
}

func newDebugShowImageCMD() *cobra.Command {
	showCmd := &cobra.Command{
		Use:   "show-images",
		Short: "List default images",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := debug.NewDebugImagesManager()
			manager.RegistryURL = debug.DefaultSealerRegistryURL

			if err := manager.ShowDefaultImages(); err != nil {
				return err
			}
			return nil
		},
	}

	return showCmd
}

func newDebugPodCommand(options *debug.DebuggerOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pod",
		Short: "Debug pod or container",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			debugger := debug.NewDebugger(options)
			debugger.AdminKubeConfigPath = common.KubeAdminConf
			debugger.Type = debug.TypeDebugPod
			debugger.Motd = debug.SealerDebugMotd

			imager := debug.NewDebugImagesManager()

			if err := debugger.CompleteAndVerifyOptions(cmd, args, imager); err != nil {
				return err
			}
			str, err := debugger.Run()
			if err != nil {
				return err
			}
			if len(str) != 0 {
				fmt.Println("The debug ID:", str)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&options.TargetContainer, "container", "c", "", "The container to be debugged.")

	return cmd
}

func newDebugNodeCommand(options *debug.DebuggerOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Debug node",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			debugger := debug.NewDebugger(options)
			debugger.AdminKubeConfigPath = common.KubeAdminConf
			debugger.Type = debug.TypeDebugNode
			debugger.Motd = debug.SealerDebugMotd

			imager := debug.NewDebugImagesManager()

			if err := debugger.CompleteAndVerifyOptions(cmd, args, imager); err != nil {
				return err
			}
			str, err := debugger.Run()
			if err != nil {
				return err
			}
			if len(str) != 0 {
				fmt.Println("The debug ID:", str)
			}

			return nil
		},
	}

	return cmd
}
