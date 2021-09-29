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
	"strconv"
	"strings"
	"time"

	"github.com/alibaba/sealer/test/suites/apply"
	"github.com/alibaba/sealer/test/suites/image"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("sealer apply", func() {
	Context("start apply", func() {
		rawClusterFilePath := apply.GetRawClusterFilePath()
		rawCluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
		rawCluster.Spec.Image = settings.TestImageName
		BeforeEach(func() {
			if rawCluster.Spec.Image != settings.TestImageName {
				//rawCluster imageName updated to customImageName
				rawCluster.Spec.Image = settings.TestImageName
				apply.MarshalClusterToFile(rawClusterFilePath, rawCluster)
			}
		})
		Context("check regular scenario that provider is ali cloud", func() {
			var tempFile string
			BeforeEach(func() {
				tempFile = testhelper.CreateTempFile()
			})

			AfterEach(func() {
				apply.DeleteClusterByFile(settings.GetClusterWorkClusterfile(rawCluster.Name))
				testhelper.RemoveTempFile(tempFile)
				testhelper.DeleteFileLocally(settings.GetClusterWorkClusterfile(rawCluster.Name))
			})

			It("init, scale up, scale down, clean up", func() {
				// 1,init cluster to 2 nodes and write to disk
				By("start to init cluster")
				sess, err := testhelper.Start(apply.SealerApplyCmd(rawClusterFilePath))
				testhelper.CheckErr(err)
				testhelper.CheckExit0(sess, settings.MaxWaiteTime)
				apply.CheckNodeNumLocally(2)

				result := testhelper.GetFileDataLocally(settings.GetClusterWorkClusterfile(rawCluster.Name))
				err = testhelper.WriteFile(tempFile, []byte(result))
				testhelper.CheckErr(err)

				//2,scale up cluster to 6 nodes and write to disk
				By("Use join command to add 3master and 3node for scale up cluster in cloud mode", func() {
					apply.SealerJoin(strconv.Itoa(2), strconv.Itoa(2))
					apply.CheckNodeNumLocally(6)
				})

				result = testhelper.GetFileDataLocally(settings.GetClusterWorkClusterfile(rawCluster.Name))
				err = testhelper.WriteFile(tempFile, []byte(result))
				testhelper.CheckErr(err)
				usedCluster := apply.LoadClusterFileFromDisk(tempFile)

				//3,scale down cluster to 4 nodes and write to disk
				By("start to scale down cluster")
				usedCluster.Spec.Nodes.Count = "1"
				usedCluster.Spec.Masters.Count = "3"
				apply.WriteClusterFileToDisk(usedCluster, tempFile)
				sess, err = testhelper.Start(apply.SealerApplyCmd(tempFile))
				testhelper.CheckErr(err)
				testhelper.CheckExit0(sess, settings.MaxWaiteTime)
				apply.CheckNodeNumLocally(4)

			})

		})

		Context("check regular scenario that provider is container", func() {
			tempFile := testhelper.CreateTempFile()
			BeforeEach(func() {
				rawCluster.Spec.Provider = settings.CONTAINER
				apply.MarshalClusterToFile(tempFile, rawCluster)
				apply.CheckDockerAndSwapOff()
			})

			AfterEach(func() {
				apply.DeleteClusterByFile(settings.GetClusterWorkClusterfile(rawCluster.Name))
				testhelper.RemoveTempFile(tempFile)
				testhelper.DeleteFileLocally(settings.GetClusterWorkClusterfile(rawCluster.Name))
			})

			It("init, scale up, scale down, clean up", func() {
				// 1,init cluster to 2 nodes and write to disk
				By("start to init cluster")
				sess, err := testhelper.Start(apply.SealerApplyCmd(tempFile))
				testhelper.CheckErr(err)
				testhelper.CheckExit0(sess, settings.MaxWaiteTime)
				apply.CheckNodeNumLocally(2)

				result := testhelper.GetFileDataLocally(settings.GetClusterWorkClusterfile(rawCluster.Name))
				err = testhelper.WriteFile(tempFile, []byte(result))
				testhelper.CheckErr(err)

				//2,scale up cluster to 6 nodes and write to disk
				By("Use join command to add 2master and 1node for scale up cluster in cloud mode", func() {
					apply.SealerJoin(strconv.Itoa(2), strconv.Itoa(1))
					apply.CheckNodeNumLocally(5)
				})

				result = testhelper.GetFileDataLocally(settings.GetClusterWorkClusterfile(rawCluster.Name))
				err = testhelper.WriteFile(tempFile, []byte(result))
				testhelper.CheckErr(err)
				usedCluster := apply.LoadClusterFileFromDisk(tempFile)

				//3,scale down cluster to 4 nodes and write to disk
				By("start to scale down cluster")
				usedCluster.Spec.Nodes.Count = "1"
				usedCluster.Spec.Masters.Count = "3"
				apply.WriteClusterFileToDisk(usedCluster, tempFile)
				sess, err = testhelper.Start(apply.SealerApplyCmd(tempFile))
				testhelper.CheckErr(err)
				testhelper.CheckExit0(sess, settings.MaxWaiteTime)
				apply.CheckNodeNumLocally(4)
				image.DoImageOps(settings.SubCmdRmiOfSealer, settings.TestImageName)
			})

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
				rawCluster.Spec.Provider = settings.AliCloud
				usedCluster := apply.CreateAliCloudInfraAndSave(rawCluster, tempFile)
				defer apply.CleanUpAliCloudInfra(usedCluster)
				sshClient := testhelper.NewSSHClientByCluster(usedCluster)
				testhelper.CheckFuncBeTrue(func() bool {
					err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
					return err == nil
				}, settings.MaxWaiteTime)

				By("start to init cluster")
				apply.GenerateClusterfile(tempFile)
				apply.SendAndApplyCluster(sshClient, tempFile)
				apply.CheckNodeNumWithSSH(sshClient, 2)

				By("Use join command to add 3master and 3node for scale up cluster in baremetal mode", func() {
					usedCluster.Spec.Nodes.Count = "3"
					usedCluster.Spec.Masters.Count = "3"
					usedCluster = apply.CreateAliCloudInfraAndSave(usedCluster, tempFile)
					//waiting for service to start
					time.Sleep(10 * time.Second)
					joinMasters := strings.Join(usedCluster.Spec.Masters.IPList[1:], ",")
					joinNodes := strings.Join(usedCluster.Spec.Nodes.IPList[1:], ",")
					//sealer join master and node
					apply.SendAndJoinCluster(sshClient, tempFile, joinMasters, joinNodes)
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
})
