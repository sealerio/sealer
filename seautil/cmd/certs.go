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
	"os"

	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/pkg/cert"
)

type Flag struct {
	AltNames     []string
	NodeName     string
	ServiceCIDR  string
	NodeIP       string
	DNSDomain    string
	CertPath     string
	CertEtcdPath string
}

var config *Flag

// certsCmd represents the certs command
var certsCmd = &cobra.Command{
	Use:   "certs",
	Short: "generate kubernetes certes",
	Long:  `seautil cert --node-ip 192.168.0.2 --node-name master1 --dns-domain aliyun.com --alt-names aliyun.local`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cert.GenerateCert(config.CertPath, config.CertEtcdPath, config.AltNames, config.NodeIP, config.NodeName, config.ServiceCIDR, config.DNSDomain)
		if err != nil {
			logger.Error(err)
			os.Exit(-1)
		}
	},
}

func init() {
	config = &Flag{}
	rootCmd.AddCommand(certsCmd)

	certsCmd.Flags().StringSliceVar(&config.AltNames, "alt-names", []string{}, "like sealyun.com or 10.103.97.2")
	certsCmd.Flags().StringVar(&config.NodeName, "node-name", "", "like master0")
	certsCmd.Flags().StringVar(&config.ServiceCIDR, "service-cidr", "", "like 10.103.97.2/24")
	certsCmd.Flags().StringVar(&config.NodeIP, "node-ip", "", "like 10.103.97.2")
	certsCmd.Flags().StringVar(&config.DNSDomain, "dns-domain", "cluster.local", "cluster dns domain")
	certsCmd.Flags().StringVar(&config.CertPath, "cert-path", "/etc/kubernetes/pki", "kubernetes cert file path")
	certsCmd.Flags().StringVar(&config.CertEtcdPath, "cert-etcd-path", "/etc/kubernetes/pki/etcd", "kubernetes etcd cert file path")
}
