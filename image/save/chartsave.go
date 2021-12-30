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

package save

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

const (
	CHARTDIR = "/charts"
)

var settings = cli.New()

//"cs" full name: chart save
func (cs *DefaultSaver) SaveCharts(charts []chart) error {
	var repoToCharts = make(map[string][]string)
	os.MkdirAll(cs.rootdir+CHARTDIR, common.FileMode0644)

	for _, chart := range charts {
		repoToCharts[chart.repo] = append(repoToCharts[chart.repo], chart.name)
	}

	for repo, charts := range repoToCharts {
		cs.saveCharts(repo, charts)
	}

	cs.registerCharts(cs.rootdir + CHARTDIR)
	return nil
}

func (cs *DefaultSaver) saveCharts(repo string, charts []string) {
	client := action.NewPullWithOpts(action.WithConfig(nil))
	client.Settings = settings
	client.ChartPathOptions.RepoURL = repo
	client.DestDir = cs.rootdir + CHARTDIR
	if client.Version == "" && client.Devel {
		debug("setting version to >0.0.0-0")
		client.Version = ">0.0.0-0"
	}

	// if err := checkOCI(args[0]); err != nil {
	// 	panic(err)
	// }

	for i := 0; i < len(charts); i++ {
		output, err := client.Run(charts[i])
		if err != nil {
			panic(err)
		}
		fmt.Fprint(os.Stdout, output)
	}
}

func debug(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func (cs *DefaultSaver) registerCharts(dir string) error {
	path, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	out := filepath.Join(path, "index.yaml")

	i, err := repo.IndexDirectory(path, "")
	if err != nil {
		return err
	}

	i.SortEntries()
	return i.WriteFile(out, 0644)
}
