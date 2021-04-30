package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/logger"
)

type ApplyFlag struct {
	ClusterFile string
}

var applyFlag *ApplyFlag

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "apply a kubernetes cluster",
	Long:  `seautil apply -f cluster.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		applier := apply.NewApplierFromFile(applyFlag.ClusterFile)
		err := applier.Apply()
		if err != nil {
			logger.Error(err)
			os.Exit(-1)
		}
	},
}

func init() {
	applyFlag = &ApplyFlag{}
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&applyFlag.ClusterFile, "clusterfile", "f", "", "cluster file filepath")
}
