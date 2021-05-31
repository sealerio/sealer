package test

import (
	"fmt"
	"github.com/alibaba/sealer/test/suites/apply"
	"github.com/alibaba/sealer/test/suites/registry"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	"github.com/alibaba/sealer/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer apply", func() {
	Context("start apply", func() {
		BeforeEach(func() {
			registry.Login()
		})
		AfterEach(func() {
			registry.Logout()
		})
		Context("check if cluster file exist", func() {
			Context("if exist", func() {
				rawClusterFilePath := apply.GetRawClusterFilePath()
				rawCluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
				usedClusterFilePath := testhelper.GetUsedClusterFilePath(rawCluster.Name)

				Context("check regular scenario that provider is ali cloud", func() {
					AfterEach(func() {
						apply.DeleteCluster(usedClusterFilePath)
					})
					It("scale up and scale down", func() {
						// 1,apply a test cluster with 2 nodes
						sess, err := testhelper.Start(apply.SealerApplyCmd(rawClusterFilePath))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						Expect(apply.GetClusterNodes()).Should(Equal(2))

						// 2,scale up cluster to 6 nodes and write to disk
						usedCluster := apply.LoadClusterFileFromDisk(usedClusterFilePath)
						usedCluster.Spec.Nodes.Count = "3"
						usedCluster.Spec.Masters.Count = "3"
						apply.WriteClusterFileToDisk(usedCluster, usedClusterFilePath)
						sess, err = testhelper.Start(apply.SealerApplyCmd(usedClusterFilePath))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						Expect(apply.GetClusterNodes()).Should(Equal(6))

						// 3,scale down cluster to 4 nodes and write to disk
						usedCluster = apply.LoadClusterFileFromDisk(usedClusterFilePath)
						usedCluster.Spec.Nodes.Count = "1"
						usedCluster.Spec.Masters.Count = "3"
						apply.WriteClusterFileToDisk(usedCluster, usedClusterFilePath)
						sess, err = testhelper.Start(apply.SealerApplyCmd(usedClusterFilePath))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						Expect(apply.GetClusterNodes()).Should(Equal(4))
					})

				})

				Context("check regular scenario that provider is bare metal", func() {
					AfterEach(func() {
						//delete infra
						apply.DeleteCluster(usedClusterFilePath)
					})

					It("scale up and scale down", func() {
						// 1,apply a remote cluster with 2 nodes and prepare ssh client
						sess, err := testhelper.Start(apply.SealerApplyCmd(rawClusterFilePath))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						Expect(apply.GetClusterNodes()).Should(Equal(2))
						usedCluster := apply.LoadClusterFileFromDisk(usedClusterFilePath)
						sshClient := testhelper.NewSSHClientByCluster(usedCluster)
						usedCluster.Spec.Provider = settings.BAREMETAL
						remoteApplyCmd := apply.SealerApplyCmd(usedClusterFilePath)
						// 2,scale up cluster to 6 nodes and write to disk
						usedCluster.Spec.Nodes.Count = "3"
						usedCluster.Spec.Masters.Count = "3"
						apply.WriteClusterFileToDisk(usedCluster, usedClusterFilePath)
						err = sshClient.SSH.Copy(sshClient.RemoteHostIP, usedClusterFilePath, usedClusterFilePath)
						Expect(err).NotTo(HaveOccurred())
						err = sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, remoteApplyCmd)
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						Expect(apply.GetClusterNodes()).Should(Equal(6))

						// 3,scale down cluster to 4 nodes and write to disk
						usedCluster.Spec.Nodes.Count = "1"
						usedCluster.Spec.Masters.Count = "3"
						apply.WriteClusterFileToDisk(usedCluster, usedClusterFilePath)
						err = sshClient.SSH.Copy(sshClient.RemoteHostIP, usedClusterFilePath, usedClusterFilePath)
						Expect(err).NotTo(HaveOccurred())
						err = sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, remoteApplyCmd)
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						Expect(apply.GetClusterNodes()).Should(Equal(4))
						// 4,delete k8s cluster:run apply delete remotely
						deleteCmd := fmt.Sprintf("sealer delete -f %s", usedClusterFilePath)
						err = sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, deleteCmd)
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						//reset provider for cleaning up
						usedCluster.Spec.Provider = settings.AliCloud
						apply.WriteClusterFileToDisk(usedCluster, usedClusterFilePath)
					})

				})

				Context("check abnormal scenario that no need to delete cluster", func() {
					var tempFile string
					BeforeEach(func() {
						tempFile = testhelper.CreateTempFile()
					})

					AfterEach(func() {
						testhelper.RemoveTempFile(tempFile)
					})

					It("empty content of cluster file", func() {
						sess, err := testhelper.Start(apply.SealerApplyCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.DefaultWaiteTime).ShouldNot(Exit(0))
					})

					It("invalid content of cluster file", func() {
						err := utils.WriteFile(tempFile, []byte("i love sealer!"))
						Expect(err).NotTo(HaveOccurred())
						sess, err := testhelper.Start(apply.SealerApplyCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.DefaultWaiteTime).ShouldNot(Exit(0))
					})

					It("invalid provider of cluster file", func() {
						rawCluster.Spec.Provider = "sealer"
						apply.WriteClusterFileToDisk(rawCluster, tempFile)
						sess, err := testhelper.Start(apply.SealerApplyCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.DefaultWaiteTime).ShouldNot(Exit(2))
					})

					It("invalid images name of cluster file", func() {
						rawCluster.Spec.Image = "FakeImage"
						apply.WriteClusterFileToDisk(rawCluster, tempFile)
						sess, err := testhelper.Start(apply.SealerApplyCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.DefaultWaiteTime).ShouldNot(Exit(0))
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
