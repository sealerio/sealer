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

	"github.com/alibaba/sealer/test/testhelper/settings"

	"github.com/onsi/gomega"

	"github.com/alibaba/sealer/test/testhelper"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

func getFixtures() string {
	pwd := testhelper.GetPwd()
	return filepath.Join(pwd, "suites", "apply", "fixtures")
}

func GetRawClusterFilePath() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "cluster_file_for_test.yaml")
}

func DeleteCluster(clusterFile string) {
	cmd := fmt.Sprintf("%s delete -f %s", settings.DefaultSealerBin, clusterFile)
	testhelper.RunCmdAndCheckResult(cmd, 0)
}

func WriteClusterFileToDisk(cluster *v1.Cluster, clusterFilePath string) {
	gomega.Expect(cluster).NotTo(gomega.BeNil())
	err := utils.MarshalYamlToFile(clusterFilePath, cluster)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func LoadClusterFileFromDisk(clusterFilePath string) *v1.Cluster {
	var cluster v1.Cluster
	err := utils.UnmarshalYamlFile(clusterFilePath, &cluster)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cluster).NotTo(gomega.BeNil())
	return &cluster
}

func GetClusterNodes() int {
	client, err := testhelper.NewClientSet()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	nodes, err := testhelper.ListNodes(client)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return len(nodes.Items)
}

func CheckClusterPods() int {
	client, err := testhelper.NewClientSet()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	pods, err := testhelper.ListNodes(client)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return len(pods.Items)
}

func SealerApplyCmd(clusterFile string) string {
	return fmt.Sprintf("%s apply -f %s", settings.DefaultSealerBin, clusterFile)
}
