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
	"strconv"

	"github.com/alibaba/sealer/test/suites/build"
	"github.com/alibaba/sealer/test/suites/image"
	"github.com/alibaba/sealer/test/suites/registry"
	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("sealer image", func() {
	Context("pull image", func() {

		It(fmt.Sprintf("pull image %s", settings.TestImageName), func() {
			image.DoImageOps(settings.SubCmdListOfSealer, settings.TestImageName)
			beforeTestEnvMd5 := image.GetEnvDirMd5()
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)
			Expect(build.CheckIsImageExist(settings.TestImageName)).Should(BeTrue())
			By("show image metadata", func() {
				testhelper.RunCmdAndCheckResult(fmt.Sprintf("%s inspect %s", settings.DefaultSealerBin, settings.TestImageName), 0)
			})

			By("show image default Clusterfile", func() {
				testhelper.RunCmdAndCheckResult(fmt.Sprintf("%s inspect -c %s", settings.DefaultSealerBin, settings.TestImageName), 0)
			})
			tagImageNames := []string{
				"e2eimage_test:latest",
				"e2eimage_test:v0.0.1",
				"sealer-io/e2eimage_test:v0.0.2",
				"registry.cn-qingdao.aliyuncs.com/sealer-io/e2eimage_test:v0.0.3",
			}
			By("tag by image name", func() {
				image.TagImageList(settings.TestImageName, tagImageNames)
				image.DoImageOps(settings.SubCmdListOfSealer, "")
				image.RemoveImageList(tagImageNames)
			})

			By("tag by image id", func() {
				imageID := image.GetImageID(settings.TestImageName)
				image.TagImageList(imageID, tagImageNames)
				image.DoImageOps(settings.SubCmdListOfSealer, "")
				image.RemoveImageList(tagImageNames)
			})

			By("remove tag image", func() {
				tagImageName := "e2e_images_test:v0.3"
				image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)

				beforeEnvMd5 := image.GetEnvDirMd5()
				By(fmt.Sprintf("beforeEnvMd5 is %s", beforeEnvMd5))
				Expect(beforeEnvMd5).NotTo(Equal(""))
				image.TagImages(settings.TestImageName, tagImageName)
				Expect(build.CheckIsImageExist(tagImageName)).Should(BeTrue())
				image.DoImageOps(settings.SubCmdRmiOfSealer, tagImageName)
				Expect(build.CheckIsImageExist(tagImageName)).ShouldNot(BeTrue())

				afterEnvMd5 := image.GetEnvDirMd5()
				By(fmt.Sprintf("afterEnvMd5 is %s", afterEnvMd5))
				Expect(afterEnvMd5).To(Equal(beforeEnvMd5))
			})

			By("force remove image", func() {
				Expect(build.CheckIsImageExist(settings.TestImageName)).Should(BeTrue())
				testImageName := "image_test:v0.0"
				for i := 1; i <= 5; i++ {
					image.TagImages(settings.TestImageName, testImageName+strconv.Itoa(i))
					image.DoImageOps(settings.SubCmdListOfSealer, settings.TestImageName)
					Expect(build.CheckIsImageExist(testImageName + strconv.Itoa(i))).Should(BeTrue())
				}
				image.DoImageOps(settings.SubCmdForceRmiOfSealer, settings.TestImageName)
				Expect(build.CheckIsImageExist(settings.TestImageName)).ShouldNot(BeTrue())
				Expect(build.CheckIsImageExist(testImageName)).ShouldNot(BeTrue())
				afterEnvMd5 := image.GetEnvDirMd5()
				By(fmt.Sprintf("afterEnvMd5 is %s", afterEnvMd5))
				Expect(afterEnvMd5).To(Equal(beforeTestEnvMd5))
			})
		})

		faultImageNames := []string{
			fmt.Sprintf("%s/%s:latest", settings.DefaultImageName, settings.DefaultImageRepo),
			fmt.Sprintf("%s:latest", settings.DefaultImageDomain),
			fmt.Sprintf("%s:latest", settings.DefaultImageRepo),
		}

		for _, faultImageName := range faultImageNames {
			faultImageName := faultImageName
			It(fmt.Sprintf("pull fault image %s", faultImageName), func() {
				sess, err := testhelper.Start(fmt.Sprintf("%s %s %s", settings.DefaultSealerBin, settings.SubCmdPullOfSealer, faultImageName))
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess, settings.MaxWaiteTime).ShouldNot(Exit(0))
				Expect(build.CheckIsImageExist(faultImageName)).ShouldNot(BeTrue())
			})
		}

	})

	Context("remove image", func() {
		It(fmt.Sprintf("remove image %s", settings.TestImageName), func() {
			image.DoImageOps(settings.SubCmdListOfSealer, "")

			beforeEnvMd5 := image.GetEnvDirMd5()
			By(fmt.Sprintf("beforeEnvMd5 is %s", beforeEnvMd5))
			Expect(beforeEnvMd5).NotTo(Equal(""))
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)
			Expect(build.CheckIsImageExist(settings.TestImageName)).Should(BeTrue())
			image.DoImageOps(settings.SubCmdRmiOfSealer, settings.TestImageName)
			Expect(build.CheckIsImageExist(settings.TestImageName)).ShouldNot(BeTrue())
			afterEnvMd5 := image.GetEnvDirMd5()
			By(fmt.Sprintf("afterEnvMd5 is %s", afterEnvMd5))
			Expect(beforeEnvMd5).To(Equal(afterEnvMd5))
		})

	})

	Context("push image", func() {
		BeforeEach(func() {
			registry.Login()
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)
		})
		AfterEach(func() {
			registry.Logout()
			image.DoImageOps(settings.SubCmdForceRmiOfSealer, settings.TestImageName)
		})
		pushImageNames := []string{
			"registry.cn-qingdao.aliyuncs.com/sealer-io/e2e_image_test:v0.01",
			"sealer-io/e2e_image_test:v0.01",
			"e2e_image_test:v0.01",
		}

		for _, pushImage := range pushImageNames {
			pushImage := pushImage
			It(fmt.Sprintf("push image %s", pushImage), func() {
				image.TagImages(settings.TestImageName, pushImage)
				image.DoImageOps(settings.SubCmdPushOfSealer, pushImage)
			})
		}
	})
})
