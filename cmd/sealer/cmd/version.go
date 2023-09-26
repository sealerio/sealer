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
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/sealerio/sealer/pkg/version"
)

var (
	shortPrint bool
	output     string
)
var sealerErr error

func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Print version info",
		Args:    cobra.NoArgs,
		Example: `sealer version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate validates the provided options.
			if output != "" && output != "yaml" && output != "json" {
				return fmt.Errorf("output format must be yaml or json")
			}
			if shortPrint {
				fmt.Println(version.Get().String())
				return nil
			}
			return PrintInfo()
		},
	}
	versionCmd.Flags().BoolVar(&shortPrint, "short", false, "If true, print just the version number.")
	versionCmd.Flags().StringVarP(&output, "output", "o", "yaml", "choose `yaml` or `json` format to print version info")
	return versionCmd
}

func PrintInfo() error {
	OutputInfo := &version.Output{}
	OutputInfo.SealerVersion = version.Get()

	if err := PrintToStd(OutputInfo); err != nil {
		return err
	}
	//TODO!
	// missinfo := []string{}
	// if OutputInfo.KubernetesVersion == nil {
	// 	missinfo = append(missinfo, "kubernetes version")
	// }
	// if OutputInfo.CriRuntimeVersion == nil {
	// 	missinfo = append(missinfo, "cri runtime version")
	// }
	// if OutputInfo.KubernetesVersion == nil || OutputInfo.CriRuntimeVersion == nil {
	// 	fmt.Printf("WARNING: Failed to get %s.\nCheck kubernetes status or use command \"sealer run\" to launch kubernetes\n", strings.Join(missinfo, " and "))
	// }
	// if OutputInfo.K0sVersion == nil {
	// 	fmt.Println("WARNING: Failed to get k0s version.\nCheck k0s status or use command \"sealer run\" to launch k0s\n")
	// }
	// if OutputInfo.K3sVersion == nil {
	// 	fmt.Println("WARNING: Failed to get k3s version.\nCheck k3s status or use command \"sealer run\" to launch k3s\n")
	// }
	return nil
}

func PrintToStd(OutputInfo *version.Output) error {
	var (
		marshalled []byte
		err        error
	)
	switch output {
	case "yaml":
		marshalled, err = yaml.Marshal(&OutputInfo)
		if err != nil {
			return fmt.Errorf("fail to marshal yaml: %w", err)
		}
		fmt.Println(string(marshalled))
	case "json":
		marshalled, err = json.Marshal(&OutputInfo)
		if err != nil {
			return fmt.Errorf("fail to marshal json: %w", err)
		}
		fmt.Println(string(marshalled))
	default:
		// There is a bug in the program if we hit this case.
		// However, we follow a policy of never panicking.
		return fmt.Errorf("versionOptions were not validated: --output=%q should have been rejected", output)
	}
	return sealerErr
}
