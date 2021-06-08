package cmd

import (
	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/cert"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/spf13/cobra"
	"os"
)

var joinArgs *common.RunArgs

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "join node to cluster",
	Example: `
join to default cluster:
	sealer join --master x.x.x.x --node x.x.x.x
join to cluster by cloud provider, just set the number of masters or nodes:
	sealer join --master 2 --node 3
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Lstat(clusterFile);err != nil {
			logger.Error(clusterFile, err)
			os.Exit(1)
		}
		applier := apply.JoinApplierFromArgs(clusterFile, joinArgs)
		if applier == nil {
			os.Exit(1)
		}
		if err := applier.Apply(); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	},
}

func init() {
	runArgs = &common.RunArgs{}
	rootCmd.AddCommand(joinCmd)
	joinCmd.Flags().StringVarP(&joinArgs.Masters, "masters", "m", "", "set Count or IPList to masters")
	joinCmd.Flags().StringVarP(&joinArgs.Nodes, "nodes", "n", "", "set Count or IPList to nodes")
	joinCmd.Flags().StringVarP(&joinArgs.User, "user", "u", "root", "set baremetal server username")
	joinCmd.Flags().StringVarP(&joinArgs.Password, "passwd", "p", "", "set cloud provider or baremetal server password")
	joinCmd.Flags().StringVarP(&joinArgs.Pk, "pk", "", cert.GetUserHomeDir()+"/.ssh/id_rsa", "set baremetal server private key")
	joinCmd.Flags().StringVarP(&joinArgs.PkPassword, "pk-passwd", "", "", "set baremetal server  private key password")
	joinCmd.Flags().StringVarP(&joinArgs.Interface, "interface", "i", "", "set default network interface name")
	joinCmd.Flags().StringVarP(&clusterFile, "Clusterfile", "f", cert.GetUserHomeDir()+"/.sealer/my-cluster/Clusterfile", "apply a kubernetes cluster")
}
