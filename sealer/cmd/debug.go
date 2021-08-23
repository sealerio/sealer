package cmd

import (
	debugger "github.com/alibaba/sealer/debug"

	"github.com/spf13/cobra"
)

var debugOptions = debugger.NewDebugOptions()

var debug = &cobra.Command{
	Use:		"debug",
	Short:		"Creating debugging sessions for pods and nodes",
	Long:		"",
	Example:	"",
}

func init() {
	rootCmd.AddCommand(debug)

	debug.AddCommand(debugger.NewDebugImages())
	debug.AddCommand(debugger.NewDebugPod(debugOptions))
	debug.AddCommand(debugger.NewDebugNode(debugOptions))
	debug.AddCommand(debugger.NewDebugClean())

	debug.PersistentFlags().StringVar(&debugOptions.Image, "image", debugOptions.Image, "Container image to use for debug container.")
	debug.PersistentFlags().StringVar(&debugOptions.DebugContainerName, "name", debugOptions.DebugContainerName, "Container name to use for debug container.")
	debug.PersistentFlags().StringSliceVar(&debugOptions.CheckList, "check-list", debugOptions.CheckList, "Check items, such as network、volume.")
	debug.PersistentFlags().StringVarP(&debugOptions.Namespace, "namespace", "n", debugOptions.Namespace, "Namespace of Pod.")
	debug.PersistentFlags().BoolVarP(&debugOptions.Interactive, "stdin", "i", debugOptions.Interactive, "Keep stdin open on the container, even if nothing is attached.")
	debug.PersistentFlags().BoolVarP(&debugOptions.TTY, "tty", "t", debugOptions.TTY, "Allocate a TTY for the debugging container.")
	debug.PersistentFlags().StringToStringP("env", "e", nil, "Environment variables to set in the container.")
}