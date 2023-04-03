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
	"github.com/labring/lvscare/care"
	"github.com/spf13/cobra"
)

var Ipvs care.LvsCare

// NewIpvsCmd ipvsCmd represents the ipvs command
func NewIpvsCmd() *cobra.Command {
	ipvsCmd := &cobra.Command{
		Use:   "ipvs",
		Short: "seautil create or care local ipvs LB",
		Long: `create ipvs rules: seautil ipvs --vs 10.1.1.2:6443 --rs 192.168.0.2:6443 --rs 192.168.0.3:6443 --health-path /healthz --health-schem https --run-once
clean ipvs rules: seautil ipvs clean`,
		Run: func(cmd *cobra.Command, args []string) {
			Ipvs.VsAndRsCare()
		},
	}

	ipvsCmd.Flags().BoolVar(&Ipvs.RunOnce, "run-once", false, "run once mode")
	ipvsCmd.Flags().BoolVarP(&Ipvs.Clean, "clean", "c", true, " clean Vip ipvs rule before join node, if Vip has no ipvs rule do nothing.")
	ipvsCmd.Flags().StringVar(&Ipvs.VirtualServer, "vs", "", "virtual server like 10.54.0.2:6443")
	ipvsCmd.Flags().StringSliceVar(&Ipvs.RealServer, "rs", []string{}, "virtual server like 192.168.0.2:6443")
	ipvsCmd.Flags().StringVar(&Ipvs.HealthPath, "health-path", "/healthz", "health check path")
	ipvsCmd.Flags().StringVar(&Ipvs.HealthSchem, "health-schem", "https", "health check scheme")
	ipvsCmd.Flags().Int32Var(&Ipvs.Interval, "interval", 5, "health check interval, unit is sec.")

	return ipvsCmd
}
