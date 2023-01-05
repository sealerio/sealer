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

package alpha

import (
	"context"
	"fmt"
	"strings"

	reference2 "github.com/distribution/distribution/v3/reference"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/image/reference"
	save2 "github.com/sealerio/sealer/pkg/image/save"
)

const (
	imageName = "IMAGE NAME"
)

var longNewSearchCmdDescription = ``

var exampleForSearchCmd = `sealer alpha search <imageDomain>/<imageRepo>/<imageName> ...
## default imageDomain: 'docker.io', default imageRepo: 'sealerio'
ex.:
  sealer alpha search kubernetes
`

// NewSearchCmd searchCmd represents the search command
func NewSearchCmd() *cobra.Command {
	searchCmd := &cobra.Command{
		Use:   "search",
		Short: "search ClusterImage in default registry",
		// TODO: add long description.
		Long:    longNewSearchCmdDescription,
		Example: exampleForSearchCmd,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			table := tablewriter.NewWriter(common.StdOut)
			table.SetHeader([]string{imageName, "Version", "NetWork-Plugins"})
			for _, imgName := range args {
				named, err := reference.ParseToNamed(imgName)
				if err != nil {
					return err
				}
				ns, err := save2.NewProxyRegistry(context.Background(), "", named.Domain())
				if err != nil {
					return err
				}
				rNamed, err := reference2.WithName(named.Repo())
				if err != nil {
					return errors.Wrapf(err, "repository %s does not exist", imgName)
				}
				repo, err := ns.Repository(context.Background(), rNamed)
				if err != nil {
					return err
				}
				tags, err := repo.Tags(context.Background()).All(context.Background())
				if err != nil {
					return fmt.Errorf("repository %s does not exist, err:%v", imgName, err)
				}
				for _, tag := range tags {
					if strings.Contains(tag, "-") {
						imgTag := strings.Split(tag, "-")
						table.Append([]string{named.String(), tag, imgTag[1]})
					} else {
						table.Append([]string{named.String(), tag, "calico"})
					}
				}
			}
			table.Render()
			return nil
		},
	}
	return searchCmd
}
