package cmd

import (
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/rpccall/server"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:     "daemon",
	Short:   "start a sealer daemon",
	Example: `sealer daemon`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := server.NewServer()
		if err != nil {
			logger.Fatal("failed to start sealer grpc server, err: %s", err)
		}
		logger.Error(s.Serve())
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
