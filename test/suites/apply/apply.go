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

package apply

import (
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

func getFixtures() string {
	pwd := testhelper.GetPwd()
	return filepath.Join(pwd, "suites", "apply", "fixtures")
}

func GetClusterFilePathOfRootfs() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "cluster_file_rootfs.yaml")
}

func GetClusterFileData(clusterFile string) *v1.Cluster {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterFile, cluster); err != nil {
		logger.Error("failed to unmarshal yamlFile to get clusterFile data")
		return nil
	}
	return cluster
}

func DeleteCluster(clusterName string) {
	cmd := "sealer delete -f " + fmt.Sprintf(settings.DefaultClusterFileNeedToBeCleaned, clusterName)
	testhelper.RunCmdAndCheckResult(cmd, 0)
}
