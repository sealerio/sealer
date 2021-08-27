package cmd

import (
	"github.com/alibaba/sealer/debug"

	"github.com/spf13/cobra"
)

var debugOptions = debug.NewDebugOptions()

var debugCommand = &cobra.Command{
	Use:		"debug",
	Short:		"Creating debugging sessions for pods and nodes",
}

func init() {
	rootCmd.AddCommand(debugCommand)

	debugCommand.AddCommand(debug.NewDebugShowImagesCommand())
	debugCommand.AddCommand(debug.NewDebugPodCommand(debugOptions))
	debugCommand.AddCommand(debug.NewDebugNodeCommand(debugOptions))
	debugCommand.AddCommand(debug.NewDebugCleanCommand())

	debugCommand.PersistentFlags().StringVar(&debugOptions.Image, "image", debugOptions.Image, "Container image to use for debug container.")
	debugCommand.PersistentFlags().StringVar(&debugOptions.DebugContainerName, "name", debugOptions.DebugContainerName, "Container name to use for debug container.")
	debugCommand.PersistentFlags().StringVar(&debugOptions.PullPolicy, "image-pull-policy", "IfNotPresent", "Container image pull policy, default policy is IfNotPresent.")
	debugCommand.PersistentFlags().StringSliceVar(&debugOptions.CheckList, "check-list", debugOptions.CheckList, "Check items, such as network、volume.")
	debugCommand.PersistentFlags().StringVarP(&debugOptions.Namespace, "namespace", "n", "default", "Namespace of Pod.")
	debugCommand.PersistentFlags().BoolVarP(&debugOptions.Interactive, "stdin", "i", debugOptions.Interactive, "Keep stdin open on the container, even if nothing is attached.")
	debugCommand.PersistentFlags().BoolVarP(&debugOptions.TTY, "tty", "t", debugOptions.TTY, "Allocate a TTY for the debugging container.")
	debugCommand.PersistentFlags().StringToStringP("env", "e", nil, "Environment variables to set in the container.")
}