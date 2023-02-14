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

	"github.com/sealerio/sealer/test/testhelper"
	"github.com/sealerio/sealer/test/testhelper/settings"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func DoImageOps(action, imageName string) {
	cmd := ""
	switch action {
	case "pull":
		cmd = fmt.Sprintf("%s pull %s -d", settings.DefaultSealerBin, imageName)
	case "push":
		cmd = fmt.Sprintf("%s push %s -d", settings.DefaultSealerBin, imageName)
	case "rmi":
		cmd = fmt.Sprintf("%s rmi %s -d", settings.DefaultSealerBin, imageName)
	case "images":
		cmd = fmt.Sprintf("%s images", settings.DefaultSealerBin)
	case "inspect":
		cmd = fmt.Sprintf("%s inspect %s -d", settings.DefaultSealerBin, imageName)
	}

	testhelper.RunCmdAndCheckResult(cmd, 0)
}
func TagImages(oldName, newName string) {
	cmd := fmt.Sprintf("%s tag %s %s", settings.DefaultSealerBin, oldName, newName)
	testhelper.RunCmdAndCheckResult(cmd, 0)
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
		gomega.Eventually(sess).Should(gbytes.Say(fmt.Sprintln("Login Succeeded!")))
		gomega.Eventually(sess).Should(gexec.Exit(0))
		return
	}
	sess, err := testhelper.Start(loginCmd)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(sess).ShouldNot(gbytes.Say(fmt.Sprintln("Login Succeeded!")))
	gomega.Eventually(sess).ShouldNot(gexec.Exit(0))
}
