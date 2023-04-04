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
	"fmt" //nolint:imports
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/version"
)

var shortPrint bool

func NewVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "version",
		Long:  `sealer version`,
		Run: func(cmd *cobra.Command, args []string) {
			marshalled, err := json.Marshal(version.Get())
			if err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
			if shortPrint {
				fmt.Println(version.Get().String())
			} else {
				fmt.Println(string(marshalled))
			}
		},
	}
	versionCmd.Flags().BoolVar(&shortPrint, "short", false, "If true, print just the version number.")
	return versionCmd
}
