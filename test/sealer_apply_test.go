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
	"time"

	. "github.com/onsi/ginkgo"

	"github.com/sealerio/sealer/test/suites/apply"
	"github.com/sealerio/sealer/test/suites/build"
	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/test/testhelper/settings"
)

var _ = Describe("sealer apply", func() {
	Context("start apply", func() {
		rawClusterFilePath := apply.GetRawClusterFilePath()
		rawCluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
		rawCluster.Spec.Image = settings.TestImageName
		BeforeEach(func() {
			if rawCluster.Spec.Image != settings.TestImageName {
				//rawClusterImageName updated to customImageName
				rawCluster.Spec.Image = settings.TestImageName
				apply.MarshalClusterToFile(rawClusterFilePath, rawCluster)
			}
		})

		Context("check regular scenario that provider is bare metal, executes machine is master0", func() {
			var tempFile string
			BeforeEach(func() {
				tempFile = testhelper.CreateTempFile()
			})

			AfterEach(func() {
				testhelper.RemoveTempFile(tempFile)
			})
			It("init, scale up, scale down, clean up", func() {
				By("start to prepare infra")
				cluster := rawCluster.DeepCopy()
				cluster.Spec.Provider = settings.AliCloud
				cluster.Spec.Image = build.GetTestImageName()
				cluster = apply.CreateAliCloudInfraAndSave(cluster, tempFile)
				defer apply.CleanUpAliCloudInfra(cluster)
				sshClient := testhelper.NewSSHClientByCluster(cluster)
				testhelper.CheckFuncBeTrue(func() bool {
					err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
					return err == nil
				}, settings.MaxWaiteTime)

				By("start to init cluster")
				apply.GenerateClusterfile(tempFile)
				apply.SendAndApplyCluster(sshClient, tempFile)
				apply.CheckNodeNumWithSSH(sshClient, 2)

				By("Wait for the cluster to be ready", func() {
					apply.WaitAllNodeRunningBySSH(sshClient.SSH, sshClient.RemoteHostIP)
				})

				By("Use join command to add 3master and 3node for scale up cluster in baremetal mode", func() {
					cluster.Spec.Nodes.Count = "3"
					cluster.Spec.Masters.Count = "3"
					cluster = apply.CreateAliCloudInfraAndSave(cluster, tempFile)
					//waiting for service to start
					time.Sleep(10 * time.Second)
					joinMasters := strings.Join(cluster.Spec.Masters.IPList[1:], ",")
					joinNodes := strings.Join(cluster.Spec.Nodes.IPList[1:], ",")
					//sealer join master and node
					apply.SendAndJoinCluster(sshClient, tempFile, joinMasters, joinNodes)
					//add 3 masters and 3 nodes
					apply.CheckNodeNumWithSSH(sshClient, 6)
				})

				By("start to scale down cluster")
				cluster.Spec.Nodes.Count = "1"
				cluster.Spec.Nodes.IPList = cluster.Spec.Nodes.IPList[:1]
				cluster.Spec.Masters.Count = "3"
				cluster.Spec.Provider = settings.BAREMETAL
				apply.WriteClusterFileToDisk(cluster, tempFile)
				apply.SendAndApplyCluster(sshClient, tempFile)
				apply.CheckNodeNumWithSSH(sshClient, 4)
				By("start to delete cluster")
				err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, apply.SealerDeleteCmd(tempFile))
				testhelper.CheckErr(err)
			})

		})

		Context("check regular scenario that provider is bare metal, executes machine is not master0", func() {
			var tempFile string
			BeforeEach(func() {
				tempFile = testhelper.CreateTempFile()
			})

			AfterEach(func() {
				testhelper.RemoveTempFile(tempFile)
				testhelper.DeleteFileLocally(settings.GetClusterWorkClusterfile(rawCluster.Name))
			})
			It("init, scale up, scale down, clean up", func() {
				By("start to prepare infra")
				cluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
				cluster.Spec.Provider = settings.AliCloud
				usedCluster := apply.ChangeMasterOrderAndSave(cluster, tempFile)
				defer apply.CleanUpAliCloudInfra(usedCluster)
				sshClient := testhelper.NewSSHClientByCluster(usedCluster)
				testhelper.CheckFuncBeTrue(func() bool {
					err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
					return err == nil
				}, settings.MaxWaiteTime)

				By("start to init cluster")
				apply.SendAndApplyCluster(sshClient, tempFile)
				apply.CheckNodeNumWithSSH(sshClient, 4)

				By("Wait for the cluster to be ready", func() {
					apply.WaitAllNodeRunningBySSH(sshClient.SSH, sshClient.RemoteHostIP)
				})

				By("Use join command to add 3master and 3node for scale up cluster in baremetal mode", func() {
					usedCluster.Spec.Nodes.Count = "3"
					usedCluster.Spec.Masters.Count = "3"
					usedCluster = apply.CreateAliCloudInfraAndSave(usedCluster, tempFile)
					//waiting for service to start
					time.Sleep(10 * time.Second)
					joinNodes := strings.Join(usedCluster.Spec.Nodes.IPList[1:], ",")
					//sealer join master and node
					apply.SendAndJoinCluster(sshClient, tempFile, "", joinNodes)
					//add 3 masters and 3 nodes
					apply.CheckNodeNumWithSSH(sshClient, 6)
				})

				By("start to scale down cluster")
				usedCluster.Spec.Nodes.Count = "1"
				usedCluster.Spec.Nodes.IPList = usedCluster.Spec.Nodes.IPList[:1]
				usedCluster.Spec.Masters.Count = "3"
				usedCluster.Spec.Provider = settings.BAREMETAL
				apply.WriteClusterFileToDisk(usedCluster, tempFile)
				apply.SendAndApplyCluster(sshClient, tempFile)
				apply.CheckNodeNumWithSSH(sshClient, 4)

				By("start to delete cluster")
				err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, apply.SealerDeleteCmd(tempFile))
				testhelper.CheckErr(err)
			})

		})
	})

	Context("start nydus image apply", func() {
		rawCluster := apply.LoadClusterFileFromDisk(apply.GetRawClusterFilePath())
		rawCluster.Spec.Image = settings.TestNydusImageName

		Context("check regular scenario that provider is bare metal, executes machine is master0", func() {
			var tempFile string
			BeforeEach(func() {
				tempFile = testhelper.CreateTempFile()
			})

			AfterEach(func() {
				testhelper.RemoveTempFile(tempFile)
			})
			It("init, scale up, scale down, clean up", func() {
				By("start to prepare infra")
				cluster := rawCluster.DeepCopy()
				cluster.Spec.Provider = settings.AliCloud
				cluster.Spec.Image = settings.TestNydusImageName
				cluster = apply.CreateAliCloudInfraAndSave(cluster, tempFile)
				defer apply.CleanUpAliCloudInfra(cluster)
				sshClient := testhelper.NewSSHClientByCluster(cluster)
				testhelper.CheckFuncBeTrue(func() bool {
					err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
					return err == nil
				}, settings.MaxWaiteTime)

				By("start to init cluster")
				apply.GenerateClusterfile(tempFile)
				apply.SendAndApplyCluster(sshClient, tempFile)
				apply.CheckNodeNumWithSSH(sshClient, 2)

				By("Wait for the cluster to be ready", func() {
					apply.WaitAllNodeRunningBySSH(sshClient.SSH, sshClient.RemoteHostIP)
				})

				By("Use join command to add 3master and 3node for scale up cluster in baremetal mode", func() {
					cluster.Spec.Nodes.Count = "3"
					cluster.Spec.Masters.Count = "3"
					cluster = apply.CreateAliCloudInfraAndSave(cluster, tempFile)
					//waiting for service to start
					time.Sleep(10 * time.Second)
					joinMasters := strings.Join(cluster.Spec.Masters.IPList[1:], ",")
					joinNodes := strings.Join(cluster.Spec.Nodes.IPList[1:], ",")
					//sealer join master and node
					apply.SendAndJoinCluster(sshClient, tempFile, joinMasters, joinNodes)
					//add 3 masters and 3 nodes
					apply.CheckNodeNumWithSSH(sshClient, 6)
				})

				By("start to scale down cluster")
				cluster.Spec.Nodes.Count = "1"
				cluster.Spec.Nodes.IPList = cluster.Spec.Nodes.IPList[:1]
				cluster.Spec.Masters.Count = "3"
				apply.WriteClusterFileToDisk(cluster, tempFile)
				apply.SendAndApplyCluster(sshClient, tempFile)
				apply.CheckNodeNumWithSSH(sshClient, 4)
				By("start to delete cluster")
				err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, apply.SealerDeleteCmd(tempFile))
				testhelper.CheckErr(err)
			})

		})
	})
})
