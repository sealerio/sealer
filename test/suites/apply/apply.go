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
	"strconv"
	"strings"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/infra"
	"github.com/alibaba/sealer/test/testhelper/settings"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/sealer/test/testhelper"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

func getFixtures() string {
	pwd := testhelper.GetPwd()
	return filepath.Join(pwd, "suites", "apply", "fixtures")
}

func GetRawClusterFilePath() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "cluster_file_for_test.yaml")
}

func DeleteClusterByFile(clusterFile string) {
	testhelper.RunCmdAndCheckResult(SealerDeleteCmd(clusterFile), 0)
}

func WriteClusterFileToDisk(cluster *v1.Cluster, clusterFilePath string) {
	gomega.Expect(cluster).NotTo(gomega.BeNil())
	err := testhelper.MarshalYamlToFile(clusterFilePath, cluster)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func LoadClusterFileFromDisk(clusterFilePath string) *v1.Cluster {
	var cluster v1.Cluster
	err := testhelper.UnmarshalYamlFile(clusterFilePath, &cluster)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cluster).NotTo(gomega.BeNil())
	return &cluster
}

func SealerDeleteCmd(clusterFile string) string {
	return fmt.Sprintf("%s delete -f %s", settings.DefaultSealerBin, clusterFile)
}

func SealerApplyCmd(clusterFile string) string {
	return fmt.Sprintf("%s apply -f %s", settings.DefaultSealerBin, clusterFile)
}

func CreateAliCloudInfraAndSave(cluster *v1.Cluster, clusterFile string) *v1.Cluster {
	gomega.Eventually(func() bool {
		infraManager := infra.NewDefaultProvider(cluster)
		err := infraManager.Apply()
		return err == nil
	}, settings.MaxWaiteTime).Should(gomega.BeTrue())

	//save used cluster file
	cluster.Spec.Provider = settings.BAREMETAL
	err := testhelper.MarshalYamlToFile(clusterFile, cluster)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	cluster.Spec.Provider = settings.AliCloud
	return cluster
}

func SendAndApplyCluster(sshClient *testhelper.SSHClient, clusterFile string) {
	// send tmp cluster file to remote server and run apply cmd
	gomega.Eventually(func() bool {
		err := sshClient.SSH.Copy(sshClient.RemoteHostIP, clusterFile, clusterFile)
		return err == nil
	}, settings.MaxWaiteTime).Should(gomega.BeTrue())

	gomega.Eventually(func() bool {
		err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, SealerApplyCmd(clusterFile))
		return err == nil
	}, settings.MaxWaiteTime).Should(gomega.BeTrue())
}

func CleanUpAliCloudInfra(cluster *v1.Cluster) {
	if cluster == nil {
		return
	}
	if cluster.Spec.Provider != settings.AliCloud {
		cluster.Spec.Provider = settings.AliCloud
	}

	gomega.Eventually(func() bool {
		t := metav1.Now()
		cluster.DeletionTimestamp = &t
		infraManager := infra.NewDefaultProvider(cluster)
		err := infraManager.Apply()
		return err == nil
	}, settings.MaxWaiteTime).Should(gomega.BeTrue())
}

// CheckNodeNumWithSSH check node mum of remote cluster;for bare metal apply
func CheckNodeNumWithSSH(sshClient *testhelper.SSHClient, expectNum int) {
	if sshClient == nil {
		return
	}
	cmd := "kubectl get nodes | wc -l"
	result, err := sshClient.SSH.CmdToString(sshClient.RemoteHostIP, cmd, "")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	num, err := strconv.Atoi(strings.ReplaceAll(result, "\n", ""))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(num).Should(gomega.Equal(expectNum + 1))
}

// CheckNodeNumLocally check node mum of remote cluster;for cloud apply
func CheckNodeNumLocally(expectNum int) {
	cmd := "sudo -E kubectl get nodes | wc -l"
	result, err := utils.RunSimpleCmd(cmd)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	num, err := strconv.Atoi(strings.ReplaceAll(result, "\n", ""))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(num).Should(gomega.Equal(expectNum + 1))
}
