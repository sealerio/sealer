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
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/test/suites/build"
	"github.com/sealerio/sealer/test/suites/image"
	"github.com/sealerio/sealer/test/suites/registry"
	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/test/testhelper/settings"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("sealer build", func() {
	Context("testing the content of kube file", func() {
		Context("testing lite build scenario", func() {

			BeforeEach(func() {
				registry.Login()
				liteBuildPath := filepath.Join(build.GetFixtures(), build.GetLiteBuildDir())
				err := os.Chdir(liteBuildPath)
				testhelper.CheckErr(err)
				//add From custom image name
				build.UpdateKubeFromImage(settings.TestImageName, filepath.Join(liteBuildPath, "Kubefile"))
			})
			AfterEach(func() {
				registry.Logout()
				err := os.Chdir(settings.DefaultTestEnvDir)
				testhelper.CheckErr(err)
			})

			It("with all build instruct", func() {
				imageName := build.GetTestImageName()
				cmd := build.NewArgsOfBuild().
					SetKubeFile("Kubefile").
					SetImageName(imageName).
					SetContext(".").
					SetBuildType(settings.LiteBuild).
					Build()
				sess, err := testhelper.Start(cmd)
				testhelper.CheckErr(err)
				testhelper.CheckExit0(sess, settings.MaxWaiteTime)
				// check: sealer images whether image exist
				testhelper.CheckBeTrue(build.CheckIsImageExist(imageName))
				image.DoImageOps(settings.SubCmdPushOfSealer, imageName)
			})
		})
	})

})
