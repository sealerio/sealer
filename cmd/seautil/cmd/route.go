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

	utilsnet "github.com/sealerio/sealer/utils/net"

	"github.com/spf13/cobra"
)

type RouteFlag struct {
	host      string
	gatewayIP string
}

var routeFlag *RouteFlag

func NewRouteCmd() *cobra.Command {
	routeCmd := &cobra.Command{
		Use:   "route",
		Short: "A brief description of your command",
	}
	routeFlag = &RouteFlag{}
	routeCmd.AddCommand(RouteAddCmd())
	routeCmd.AddCommand(RouteDelCmd())
	routeCmd.AddCommand(RouteCheckCmd())
	return routeCmd
}

func RouteCheckCmd() *cobra.Command {
	var checkCmd = &cobra.Command{
		Use:   "check",
		Short: "A brief description of your command",
		Long:  `seautil route check --host 192.168.56.3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			host := net.ParseIP(routeFlag.host)
			if host == nil {
				return fmt.Errorf("input host(%s) is invalid: it should be an IP format", routeFlag.host)
			}
			return utilsnet.CheckIsDefaultRoute(host)
		},
	}
	checkCmd.Flags().StringVar(&routeFlag.host, "host", "", "check host ip address is default iFace")
	return checkCmd
}

func RouteAddCmd() *cobra.Command {
	var addCmd = &cobra.Command{
		Use:   "add",
		Short: "A brief description of your command",
		Long:  `seautil route add --host 192.168.0.2 --gateway 10.0.0.2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			host := net.ParseIP(routeFlag.host)
			if host == nil {
				return fmt.Errorf("input host(%s) is invalid: it must be an IP format", routeFlag.host)
			}

			gateway := net.ParseIP(routeFlag.gatewayIP)
			if gateway == nil {
				return fmt.Errorf("input gateway(%s) is invalid: it must be an IP format", routeFlag.gatewayIP)
			}
			r := utilsnet.NewRouter(host, gateway)
			return r.SetRoute()
		},
	}
	addCmd.Flags().StringVar(&routeFlag.host, "host", "", "route host ,ex ip route add host via gateway")
	addCmd.Flags().StringVar(&routeFlag.gatewayIP, "gateway", "", "route gateway ,ex ip route add host via gateway")
	return addCmd
}

func RouteDelCmd() *cobra.Command {
	var delCmd = &cobra.Command{
		Use:   "del",
		Short: "delete router",
		Long:  `seautil route del --host 192.168.0.2 --gateway 10.0.0.2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			host := net.ParseIP(routeFlag.host)
			if host == nil {
				return fmt.Errorf("input host(%s) is invalid: it must be an IP format", routeFlag.host)
			}

			gateway := net.ParseIP(routeFlag.gatewayIP)
			if gateway == nil {
				return fmt.Errorf("input gateway(%s) is invalid: it must be an IP format", routeFlag.gatewayIP)
			}

			r := utilsnet.NewRouter(host, gateway)
			return r.DelRoute()
		},
	}
	delCmd.Flags().StringVar(&routeFlag.host, "host", "", "route host ,ex ip route del host via gateway")
	delCmd.Flags().StringVar(&routeFlag.gatewayIP, "gateway", "", "route gateway ,ex ip route del host via gateway")
	return delCmd
}
