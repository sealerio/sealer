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
		pullImageNames := []string{
			"registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9",
			"registry.cn-qingdao.aliyuncs.com/kubernetes:v1.19.9",
			"sealer-io/kubernetes:v1.19.9",
			"kubernetes:v1.19.9",
		}

		for _, imageName := range pullImageNames {
			imageName := imageName
			It(fmt.Sprintf("pull image %s", imageName), func() {
				image.DoImageOps(settings.SubCmdPullOfSealer, imageName)
				Expect(build.CheckIsImageExist(imageName)).Should(BeTrue())
				image.DoImageOps(settings.SubCmdRmiOfSealer, imageName)
				Expect(build.CheckIsImageExist(imageName)).ShouldNot(BeTrue())
			})
		}

		faultImageNames := []string{
			"registry.cn-qingdao.aliyuncs.com/sealer-io:latest",
			"registry.cn-qingdao.aliyuncs.com:latest",
			"sealer-io:latest",
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

		It("remove tag image", func() {
			tagImageName := "e2e_images_test:v0.01"
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

			image.DoImageOps(settings.SubCmdRmiOfSealer, settings.TestImageName)
			Expect(build.CheckIsImageExist(settings.TestImageName)).ShouldNot(BeTrue())
		})

		It("force remove image", func() {
			beforeEnvMd5 := image.GetEnvDirMd5()
			By(fmt.Sprintf("beforeEnvMd5 is %s", beforeEnvMd5))
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)
			Expect(build.CheckIsImageExist(settings.TestImageName)).Should(BeTrue())
			testImageName := "image_test:v0.0"
			for i := 1; i <= 10; i++ {
				image.TagImages(settings.TestImageName, testImageName+strconv.Itoa(i))
				Expect(build.CheckIsImageExist(testImageName + strconv.Itoa(i))).Should(BeTrue())
			}
			image.DoImageOps(settings.SubCmdForceRmiOfSealer, settings.TestImageName)
			Expect(build.CheckIsImageExist(settings.TestImageName)).ShouldNot(BeTrue())
			Expect(build.CheckIsImageExist(testImageName)).ShouldNot(BeTrue())
			afterEnvMd5 := image.GetEnvDirMd5()
			By(fmt.Sprintf("afterEnvMd5 is %s", afterEnvMd5))
			Expect(afterEnvMd5).To(Equal(beforeEnvMd5))
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

	Context("inspect image", func() {
		BeforeEach(func() {
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)
		})
		AfterEach(func() {
			image.DoImageOps(settings.SubCmdRmiOfSealer, settings.TestImageName)
		})
		It("show image metadata", func() {
			testhelper.RunCmdAndCheckResult(fmt.Sprintf("%s inspect %s", settings.DefaultSealerBin, settings.TestImageName), 0)
		})

		It("show image default Clusterfile", func() {
			testhelper.RunCmdAndCheckResult(fmt.Sprintf("%s inspect -c %s", settings.DefaultSealerBin, settings.TestImageName), 0)
		})
	})

	Context("tag image", func() {
		BeforeEach(func() {
			image.DoImageOps(settings.SubCmdPullOfSealer, settings.TestImageName)
		})
		AfterEach(func() {
			image.DoImageOps(settings.SubCmdForceRmiOfSealer, settings.TestImageName)
		})
		tagImageNames := []string{
			"e2eimage_test:latest",
			"e2eimage_test:v0.01",
			"sealer-io/e2eimage_test:v0.02",
			"registry.cn-qingdao.aliyuncs.com/sealer-io/e2eimage_test:v0.03",
		}
		It("tag by image name and show images list", func() {
			image.TagImageList(settings.TestImageName, tagImageNames)
			image.DoImageOps(settings.SubCmdListOfSealer, "")
		})

		It("tag by image id and show images list", func() {
			imageID := image.GetImageID(settings.TestImageName)
			image.TagImageList(imageID, tagImageNames)
			image.DoImageOps(settings.SubCmdListOfSealer, "")
		})
	})
})
