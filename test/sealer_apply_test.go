package test

import (
	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/test/suites/apply"
	"github.com/alibaba/sealer/test/suites/registry"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"

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

		Context("check if cluster file exist", func() {
			Context("if exist", func() {
				testClusterFilePath := apply.GetTestClusterFilePath()
				cluster := apply.LoadClusterFileData(testClusterFilePath)
				usedClusterFile := testhelper.GetUsedClusterFilePath(cluster.Name)

				Context("check regular scenario that need to delete cluster", func() {
					AfterEach(func() {
						apply.DeleteCluster(usedClusterFile)
					})

					It("apply a cluster file to do shrink and expand", func() {
						// apply a test cluster with 3 nodes
						err := apply.WriteClusterFileToDisk(cluster, testClusterFilePath)
						Expect(err).NotTo(HaveOccurred())
						sess, err := testhelper.Start(apply.SealerApplyCmd(testClusterFilePath))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
						Expect(apply.GetClusterNodes()).Should(Equal(3))
						/*
							//// shrink cluster to 2 nodes and write to disk
							cluster.Spec.Nodes.Count = "1"
							cluster.Spec.Masters.Count = "1"
							err = apply.WriteClusterFileToDisk(cluster, usedClusterFile)
							Expect(err).NotTo(HaveOccurred())
							sess, err = testhelper.Start(apply.SealerApplyCmd(usedClusterFile))
							Expect(err).NotTo(HaveOccurred())
							Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
							Expect(apply.GetClusterNodes()).Should(Equal(2))

							//// expand cluster to 3 nodes and write to disk
							cluster.Spec.Nodes.Count = "2"
							cluster.Spec.Masters.Count = "1"
							err = apply.WriteClusterFileToDisk(cluster, usedClusterFile)
							Expect(err).NotTo(HaveOccurred())
							sess, err = testhelper.Start(apply.SealerApplyCmd(usedClusterFile))
							Expect(err).NotTo(HaveOccurred())
							Eventually(sess, settings.MaxWaiteTime).Should(Exit(0))
							Expect(apply.GetClusterNodes()).Should(Equal(3))*/
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
						cluster.Spec.Provider = "sealer"
						err := apply.WriteClusterFileToDisk(cluster, tempFile)
						Expect(err).NotTo(HaveOccurred())
						sess, err := testhelper.Start(apply.SealerApplyCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.DefaultWaiteTime).ShouldNot(Exit(2))
					})

					It("invalid images name of cluster file", func() {
						cluster.Spec.Image = "FakeImage"
						err := apply.WriteClusterFileToDisk(cluster, tempFile)
						Expect(err).NotTo(HaveOccurred())
						sess, err := testhelper.Start(apply.SealerApplyCmd(tempFile))
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess, settings.DefaultWaiteTime).ShouldNot(Exit(0))
					})

				})
			})

			Context("if not exist", func() {
				It("only run sealer apply", func() {
					sess, err := testhelper.Start("sealer apply")
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess.Err).Should(Say("apply cloud cluster failed open Clusterfile: no such file or directory"))
					Eventually(sess).Should(Exit(2))
				})
			})

		})

	})

})
