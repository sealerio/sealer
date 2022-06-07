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

package image

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/image/store"
	"github.com/sealerio/sealer/test/suites/build"
	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/test/testhelper/settings"
	"github.com/sealerio/sealer/utils/exec"
	"github.com/sealerio/sealer/utils/platform"
)

func DoImageOps(action, imageName string) {
	cmd := ""
	switch action {
	case settings.SubCmdPullOfSealer:
		cmd = fmt.Sprintf("%s pull %s -d", settings.DefaultSealerBin, imageName)
	case settings.SubCmdPushOfSealer:
		cmd = fmt.Sprintf("%s push %s -d", settings.DefaultSealerBin, imageName)
	case settings.SubCmdRmiOfSealer:
		cmd = fmt.Sprintf("%s rmi %s -d", settings.DefaultSealerBin, imageName)
	case settings.SubCmdRunOfSealer:
		cmd = fmt.Sprintf("%s run %s -d", settings.DefaultSealerBin, imageName)
	case settings.SubCmdListOfSealer:
		cmd = fmt.Sprintf("%s images", settings.DefaultSealerBin)
	}

	testhelper.RunCmdAndCheckResult(cmd, 0)
}
func TagImages(oldName, newName string) {
	cmd := fmt.Sprintf("%s %s %s %s", settings.DefaultSealerBin, settings.SubCmdTagOfSealer, oldName, newName)
	testhelper.RunCmdAndCheckResult(cmd, 0)
}

func GetEnvDirMd5() string {
	getEnvMd5Cmd := fmt.Sprintf("sudo -E find %s -type f -print0|xargs -0 sudo md5sum|cut -d\" \" -f1|md5sum|cut -d\" \" -f1\n", filepath.Dir(common.DefaultImageRootDir))
	dirMd5, err := exec.RunSimpleCmd(getEnvMd5Cmd)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	_, err = io.WriteString(ginkgo.GinkgoWriter, getEnvMd5Cmd+dirMd5+"\n")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return dirMd5
}

func GetImageID(imageName string) string {
	is, err := store.NewDefaultImageStore()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	image, err := is.GetByName(imageName, platform.GetDefaultPlatform())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return image.Spec.ID
}

func CheckLoginResult(registryURL, username, passwd string, result bool) {
	usernameCmd, passwdCmd := "", ""
	if username != "" {
		usernameCmd = fmt.Sprintf("-u %s", username)
	}
	if passwd != "" {
		passwdCmd = fmt.Sprintf("-p %s", passwd)
	}
	loginCmd := fmt.Sprintf("%s login %s %s %s", settings.DefaultSealerBin,
		settings.RegistryURL,
		usernameCmd,
		passwdCmd)
	if result {
		sess, err := testhelper.Start(loginCmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Eventually(sess).Should(gbytes.Say(fmt.Sprintf("%s login %s success", username, registryURL)))
		gomega.Eventually(sess).Should(gexec.Exit(0))
		return
	}
	sess, err := testhelper.Start(loginCmd)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(sess).ShouldNot(gbytes.Say(fmt.Sprintf("%s login %s success", username, registryURL)))
	gomega.Eventually(sess).ShouldNot(gexec.Exit(0))
}

func TagImageList(imageNameOrID string, tagImageNames []string) {
	for _, tagImageName := range tagImageNames {
		tagImageName := tagImageName
		TagImages(imageNameOrID, tagImageName)
		gomega.Expect(build.CheckIsImageExist(settings.TestImageName)).Should(gomega.BeTrue())
	}
}

func RemoveImageList(imageNameList []string) {
	for _, imageName := range imageNameList {
		removeImage := imageName
		DoImageOps(settings.SubCmdRmiOfSealer, removeImage)
	}
}
