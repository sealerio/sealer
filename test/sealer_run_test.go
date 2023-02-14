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

package test

import (
	"strings"

	"github.com/sealerio/sealer/test/suites/apply"
	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/test/testhelper/client/k8s"
	"github.com/sealerio/sealer/test/testhelper/settings"
	utilsnet "github.com/sealerio/sealer/utils/net"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("sealer run", func() {

	Context("run on container", func() {
		var tempFile string
		apply.CheckDockerAndSwapOff()
		BeforeEach(func() {
			tempFile = testhelper.CreateTempFile()
		})

		AfterEach(func() {
			testhelper.RemoveTempFile(tempFile)
		})

		It("container run", func() {
			rawCluster := apply.LoadClusterFileFromDisk(apply.GetRawClusterFilePath())
			rawCluster.Spec.Image = settings.TestImageName
			By("start to prepare infra")
			usedCluster := apply.CreateContainerInfraAndSave(rawCluster, tempFile)
			//defer to delete cluster
			defer func() {
				apply.CleanUpContainerInfra(usedCluster)
			}()
			sshClient := testhelper.NewSSHClientByCluster(usedCluster)
			testhelper.CheckFuncBeTrue(func() bool {
				err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
				return err == nil
			}, settings.MaxWaiteTime)

			By("start to init cluster")
			masterIPStrs := utilsnet.IPsToIPStrs(usedCluster.Spec.Masters.IPList)
			masters := strings.Join(masterIPStrs, ",")
			nodesIPStrs := utilsnet.IPsToIPStrs(usedCluster.Spec.Nodes.IPList)
			nodes := strings.Join(nodesIPStrs, ",")
			apply.SendAndRunCluster(sshClient, tempFile, masters, nodes, usedCluster.Spec.SSH.Passwd)
			client, err := k8s.NewK8sClient(sshClient)
			testhelper.CheckErr(err)
			apply.CheckNodeNumWithSSH(client, 2)

			By("Wait for the cluster to be ready", func() {
				apply.WaitAllNodeRunningBySSH(client)
			})
		})
	})
})
