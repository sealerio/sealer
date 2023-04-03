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
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/clustercert"
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

// NewCmdCert return "seautil cert" command.
func NewCmdCert() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cert",
		Short: "seautil cert experimental sub-commands",
	}
	cmd.AddCommand(NewCertGenCmd())
	cmd.AddCommand(NewCertUpdateCmd())
	return cmd
}

// NewCertGenCmd gen all kubernetes certs
func NewCertGenCmd() *cobra.Command {
	flag := new(Flag)

	// certsCmd represents the certs command
	certsCmd := &cobra.Command{
		Use:   "gen",
		Short: "generate kubernetes certs",
		Long:  `seautil cert gen --node-ip 192.168.0.2 --node-name master1 --dns-domain sealer.com --alt-names sealer.local --service-cidr 10.103.97.2/24`,
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeIP := net.ParseIP(flag.NodeIP)
			if nodeIP == nil {
				return fmt.Errorf("input --node-ip(%s) is not a valid IP format", flag.NodeIP)
			}
			return clustercert.GenerateAllKubernetesCerts(flag.CertPath, flag.CertEtcdPath, flag.NodeName, flag.ServiceCIDR, flag.DNSDomain, flag.AltNames, nodeIP)
		},
	}

	certsCmd.Flags().StringSliceVar(&flag.AltNames, "alt-names", []string{}, "like sealyun.com or 10.103.97.2")
	certsCmd.Flags().StringVar(&flag.NodeName, "node-name", "", "like master0")
	certsCmd.Flags().StringVar(&flag.ServiceCIDR, "service-cidr", "", "like 10.103.97.2/24")
	certsCmd.Flags().StringVar(&flag.NodeIP, "node-ip", "", "like 10.103.97.2")
	certsCmd.Flags().StringVar(&flag.DNSDomain, "dns-domain", "cluster.local", "cluster dns domain")
	certsCmd.Flags().StringVar(&flag.CertPath, "cert-path", clustercert.KubeDefaultCertPath, "kubernetes cert file path")
	certsCmd.Flags().StringVar(&flag.CertEtcdPath, "cert-etcd-path", clustercert.KubeDefaultCertEtcdPath, "kubernetes etcd cert file path")

	return certsCmd
}

func NewCertUpdateCmd() *cobra.Command {
	var altNames []string

	certCmd := &cobra.Command{
		Use:   "update",
		Short: "Update Kubernetes API server's cert",
		Long:  `seautil cert update --alt-names sealer.cool`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(altNames) == 0 {
				return fmt.Errorf("IP address or DNS domain needed for cert Subject Alternative Names")
			}

			err := clustercert.UpdateAPIServerCertSans(clustercert.KubeDefaultCertPath, altNames)
			if err != nil {
				return fmt.Errorf("failed to update api server's cert: %v", err)
			}
			return nil
		},
	}

	certCmd.Flags().StringSliceVar(&altNames, "alt-names", []string{}, "add DNS domain or IP in api server's cert, if it is already in the cert subject alternative names list, nothing will be changed")

	if err := certCmd.MarkFlagRequired("alt-names"); err != nil {
		logrus.Errorf("failed to init flag alt-names: %v", err)
		os.Exit(1)
	}

	return certCmd
}
