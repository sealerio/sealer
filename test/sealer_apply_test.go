// Copyright © 2021 Alibaba Group Holding Ltd.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"strings"
	"time"

	"github.com/sealerio/sealer/test/suites/apply"
	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/test/testhelper/client/k8s"
	"github.com/sealerio/sealer/test/testhelper/settings"
	utilsnet "github.com/sealerio/sealer/utils/net"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("sealer apply", func() {
	Context("start apply", func() {
		rawClusterFilePath := apply.GetRawClusterFilePath()
		rawCluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
		rawCluster.Spec.Image = settings.TestImageName
		apply.CheckDockerAndSwapOff()
		BeforeEach(func() {
			if rawCluster.Spec.Image != settings.TestImageName {
				//rawClusterImageName updated to customImageName
				rawCluster.Spec.Image = settings.TestImageName
				apply.MarshalClusterToFile(rawClusterFilePath, rawCluster)
			}
		})

		Context("check regular scenario that provider is container, executes machine is master0", func() {
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
				cluster.Spec.Provider = settings.CONTAINER
				cluster.Spec.Image = settings.TestImageName
				cluster = apply.CreateContainerInfraAndSave(cluster, tempFile)
				defer apply.CleanUpContainerInfra(cluster)
				sshClient := testhelper.NewSSHClientByCluster(cluster)
				testhelper.CheckFuncBeTrue(func() bool {
					err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
					return err == nil
				}, settings.MaxWaiteTime)

				By("start to init cluster")
				apply.GenerateClusterfile(tempFile)
				apply.SendAndApplyCluster(sshClient, tempFile)
				client, err := k8s.NewK8sClient(sshClient)
				testhelper.CheckErr(err)
				apply.CheckNodeNumWithSSH(client, 2)

				By("Wait for the cluster to be ready", func() {
					apply.WaitAllNodeRunningBySSH(client)
				})

				By("Use join command to add 1master and 2nodes for scale up cluster in container mode", func() {
					cluster.Spec.Nodes.Count = "2"
					cluster.Spec.Masters.Count = "1"
					cluster = apply.CreateContainerInfraAndSave(cluster, tempFile)
					//waiting for service to start
					time.Sleep(10 * time.Second)
					joinMastersIPStrs := utilsnet.IPsToIPStrs(cluster.Spec.Masters.IPList[1:])
					joinMasters := strings.Join(joinMastersIPStrs, ",")
					joinNodesIPStrs := utilsnet.IPsToIPStrs(cluster.Spec.Nodes.IPList[1:])
					joinNodes := strings.Join(joinNodesIPStrs, ",")
					//sealer join master and node
					apply.SendAndJoinCluster(sshClient, tempFile, joinMasters, joinNodes)
					//add 3 masters and 3 nodes
					apply.CheckNodeNumWithSSH(client, 3)
				})

				By("start to scale down cluster")
				deleteNode := cluster.Spec.Nodes.IPList[1].String()
				err = sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, nil, apply.SealerDeleteNodeCmd(deleteNode))
				testhelper.CheckErr(err)
				apply.CheckNodeNumWithSSH(client, 2)
				cluster.Spec.Nodes.Count = "1"
				cluster.Spec.Nodes.IPList = cluster.Spec.Nodes.IPList[:1]
				cluster.Spec.Masters.Count = "1"
				cluster.Spec.Provider = settings.CONTAINER
				apply.WriteClusterFileToDisk(cluster, tempFile)
				// sealer apply scale down is not supported, if supported, we can use this
				// apply.SendAndApplyCluster(sshClient, tempFile)

				By("start to delete cluster")
				err = sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, nil, apply.SealerDeleteAll())
				testhelper.CheckErr(err)
			})

		})

		// Context("check regular scenario that provider is bare metal, executes machine is not master0", func() {
		// 	var tempFile string
		// 	BeforeEach(func() {
		// 		tempFile = testhelper.CreateTempFile()
		// 	})

		// 	AfterEach(func() {
		// 		testhelper.RemoveTempFile(tempFile)
		// 		testhelper.DeleteFileLocally(settings.GetClusterWorkClusterfile(rawCluster.Name))
		// 	})
		// 	It("init, scale up, scale down, clean up", func() {
		// 		By("start to prepare infra")
		// 		cluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
		// 		cluster.Spec.Provider = settings.CONTAINER
		// 		usedCluster := apply.ChangeMasterOrderAndSave(cluster, tempFile)
		// 		defer apply.CleanUpContainerInfra(usedCluster)
		// 		sshClient := testhelper.NewSSHClientByCluster(usedCluster)
		// 		testhelper.CheckFuncBeTrue(func() bool {
		// 			err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
		// 			return err == nil
		// 		}, settings.MaxWaiteTime)

		// 		By("start to init cluster")
		// 		apply.SendAndApplyCluster(sshClient, tempFile)
		// 		apply.CheckNodeNumWithSSH(sshClient, 4)

		// 		By("Wait for the cluster to be ready", func() {
		// 			apply.WaitAllNodeRunningBySSH(sshClient.SSH, sshClient.RemoteHostIP)
		// 		})

		// 		By("Use join command to add 3master and 3node for scale up cluster in baremetal mode", func() {
		// 			usedCluster.Spec.Nodes.Count = "3"
		// 			usedCluster.Spec.Masters.Count = "3"
		// 			usedCluster = apply.CreateContainerInfraAndSave(usedCluster, tempFile)
		// 			//waiting for service to start
		// 			time.Sleep(10 * time.Second)
		// 			joinNodesIPStrs := utilsnet.IPsToIPStrs(usedCluster.Spec.Nodes.IPList[1:])
		// 			joinNodes := strings.Join(joinNodesIPStrs, ",")
		// 			//sealer join master and node
		// 			apply.SendAndJoinCluster(sshClient, tempFile, "", joinNodes)
		// 			//add 3 masters and 3 nodes
		// 			apply.CheckNodeNumWithSSH(sshClient, 6)
		// 		})

		// 		By("start to scale down cluster")
		// 		usedCluster.Spec.Nodes.Count = "1"
		// 		usedCluster.Spec.Nodes.IPList = usedCluster.Spec.Nodes.IPList[:1]
		// 		usedCluster.Spec.Masters.Count = "3"
		// 		usedCluster.Spec.Provider = settings.BAREMETAL
		// 		apply.WriteClusterFileToDisk(usedCluster, tempFile)
		// 		apply.SendAndApplyCluster(sshClient, tempFile)
		// 		apply.CheckNodeNumWithSSH(sshClient, 4)

		// 		By("start to delete cluster")
		// 		err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, apply.SealerDeleteCmd(tempFile))
		// 		testhelper.CheckErr(err)
		// 	})

		// })
	})

	// Context("start nydus image apply", func() {
	// 	rawCluster := apply.LoadClusterFileFromDisk(apply.GetRawClusterFilePath())
	// 	rawCluster.Spec.Image = settings.TestNydusImageName

	// 	Context("check regular scenario that provider is bare metal, executes machine is master0", func() {
	// 		var tempFile string
	// 		BeforeEach(func() {
	// 			tempFile = testhelper.CreateTempFile()
	// 		})

	// 		AfterEach(func() {
	// 			testhelper.RemoveTempFile(tempFile)
	// 		})
	// 		It("init, scale up, scale down, clean up", func() {
	// 			By("start to prepare infra")
	// 			cluster := rawCluster.DeepCopy()
	// 			cluster.Spec.Provider = settings.CONTAINER
	// 			cluster.Spec.Image = settings.TestNydusImageName
	// 			cluster = apply.CreateContainerInfraAndSave(cluster, tempFile)
	// 			defer apply.CleanUpContainerInfra(cluster)
	// 			sshClient := testhelper.NewSSHClientByCluster(cluster)
	// 			testhelper.CheckFuncBeTrue(func() bool {
	// 				err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
	// 				return err == nil
	// 			}, settings.MaxWaiteTime)

	// 			By("start to init cluster")
	// 			apply.GenerateClusterfile(tempFile)
	// 			apply.SendAndApplyCluster(sshClient, tempFile)
	// 			apply.CheckNodeNumWithSSH(sshClient, 2)

	// 			By("Wait for the cluster to be ready", func() {
	// 				apply.WaitAllNodeRunningBySSH(sshClient.SSH, sshClient.RemoteHostIP)
	// 			})

	// 			By("Use join command to add 3master and 3node for scale up cluster in baremetal mode", func() {
	// 				cluster.Spec.Nodes.Count = "3"
	// 				cluster.Spec.Masters.Count = "3"
	// 				cluster = apply.CreateContainerInfraAndSave(cluster, tempFile)
	// 				//waiting for service to start
	// 				time.Sleep(10 * time.Second)
	// 				joinMastersIPStrs := utilsnet.IPsToIPStrs(cluster.Spec.Masters.IPList[1:])
	// 				joinMasters := strings.Join(joinMastersIPStrs, ",")
	// 				joinNodesIPStrs := utilsnet.IPsToIPStrs(cluster.Spec.Nodes.IPList[1:])
	// 				joinNodes := strings.Join(joinNodesIPStrs, ",")
	// 				//sealer join master and node
	// 				apply.SendAndJoinCluster(sshClient, tempFile, joinMasters, joinNodes)
	// 				//add 3 masters and 3 nodes
	// 				apply.CheckNodeNumWithSSH(sshClient, 6)
	// 			})

	// 			By("start to scale down cluster")
	// 			cluster.Spec.Nodes.Count = "1"
	// 			cluster.Spec.Nodes.IPList = cluster.Spec.Nodes.IPList[:1]
	// 			cluster.Spec.Masters.Count = "3"
	// 			apply.WriteClusterFileToDisk(cluster, tempFile)
	// 			apply.SendAndApplyCluster(sshClient, tempFile)
	// 			apply.CheckNodeNumWithSSH(sshClient, 4)
	// 			By("start to delete cluster")
	// 			err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, apply.SealerDeleteCmd(tempFile))
	// 			testhelper.CheckErr(err)
	// 		})

	// 	})
	// })
})
