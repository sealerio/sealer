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
	"fmt"

	"github.com/alibaba/sealer/test/suites/apply"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer apply", func() {
	Context("start apply", func() {

		Context("check if cluster file exist", func() {
			Context("if exist", func() {
				rawClusterFilePath := apply.GetRawClusterFilePath()
				rawCluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
				Context("check regular scenario that provider is ali cloud", func() {
					var tempFile string
					BeforeEach(func() {
						tempFile = testhelper.CreateTempFile()
					})

					AfterEach(func() {
						apply.DeleteClusterByFile(testhelper.GetRootClusterFilePath(rawCluster.Name))
						testhelper.RemoveTempFile(tempFile)
						testhelper.DeleteFileLocally(testhelper.GetRootClusterFilePath(rawCluster.Name))
					})

					It("init, scale up, scale down, clean up", func() {
						// 1,init cluster to 2 nodes and write to disk
						By("start to init cluster")
						sess, err := testhelper.Start(apply.SealerApplyCmd(rawClusterFilePath))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						apply.CheckNodeNumLocally(2)

						result := testhelper.GetFileDataLocally(testhelper.GetRootClusterFilePath(rawCluster.Name))
						err = testhelper.WriteFile(tempFile, []byte(result))
						Expect(err).NotTo(HaveOccurred())
						usedCluster := apply.LoadClusterFileFromDisk(tempFile)

						// 2,scale up cluster to 6 nodes and write to disk
						By("start to scale up cluster")
						usedCluster.Spec.Nodes.Count = "3"
						usedCluster.Spec.Masters.Count = "3"
						apply.WriteClusterFileToDisk(usedCluster, tempFile)
						sess, err = testhelper.Start(apply.SealerApplyCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						apply.CheckNodeNumLocally(6)

						result = testhelper.GetFileDataLocally(testhelper.GetRootClusterFilePath(rawCluster.Name))
						err = testhelper.WriteFile(tempFile, []byte(result))
						Expect(err).NotTo(HaveOccurred())
						usedCluster = apply.LoadClusterFileFromDisk(tempFile)

						//3,scale down cluster to 4 nodes and write to disk
						By("start to scale down cluster")
						usedCluster.Spec.Nodes.Count = "1"
						usedCluster.Spec.Masters.Count = "3"
						apply.WriteClusterFileToDisk(usedCluster, tempFile)
						sess, err = testhelper.Start(apply.SealerApplyCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						apply.CheckNodeNumLocally(4)

					})

				})

				Context("check regular scenario that provider is bare metal", func() {
					var tempFile string
					BeforeEach(func() {
						tempFile = testhelper.CreateTempFile()
					})

					AfterEach(func() {
						testhelper.RemoveTempFile(tempFile)
					})
					It("init, scale up, scale down, clean up", func() {
						By("start to prepare infra")
						usedCluster := apply.CreateAliCloudInfraAndSave(rawCluster, tempFile)
						defer func() {
							apply.CleanUpAliCloudInfra(usedCluster)
						}()
						sshClient := testhelper.NewSSHClientByCluster(usedCluster)
						Eventually(func() bool {
							err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
							return err == nil
						}, settings.MaxWaiteTime).Should(BeTrue())

						By("start to init cluster")
						apply.SendAndApplyCluster(sshClient, tempFile)
						apply.CheckNodeNumWithSSH(sshClient, 2)

						By("start to scale up cluster")
						usedCluster.Spec.Nodes.Count = "3"
						usedCluster.Spec.Masters.Count = "3"
						usedCluster = apply.CreateAliCloudInfraAndSave(usedCluster, tempFile)
						apply.SendAndApplyCluster(sshClient, tempFile)
						apply.CheckNodeNumWithSSH(sshClient, 6)

						By("start to scale down cluster")
						usedCluster.Spec.Nodes.Count = "1"
						usedCluster.Spec.Nodes.IPList = usedCluster.Spec.Nodes.IPList[:1]
						usedCluster.Spec.Masters.Count = "3"
						usedCluster.Spec.Provider = settings.BAREMETAL
						apply.WriteClusterFileToDisk(usedCluster, tempFile)
						apply.SendAndApplyCluster(sshClient, tempFile)
						apply.CheckNodeNumWithSSH(sshClient, 4)
						usedCluster.Spec.Provider = settings.AliCloud
						usedCluster = apply.CreateAliCloudInfraAndSave(usedCluster, tempFile)

						By("start to delete cluster")
						err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, apply.SealerDeleteCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
					})

				})

			})

			Context("if not exist", func() {
				It("only run sealer apply", func() {
					sess, err := testhelper.Start(fmt.Sprintf("%s apply", settings.DefaultSealerBin))
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess.Err).Should(Say("apply cloud cluster failed open Clusterfile: no such file or directory"))
					Eventually(sess).Should(Exit(2))
				})
			})

		})

	})
})
