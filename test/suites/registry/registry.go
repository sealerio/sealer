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

package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/test/testhelper"
	"github.com/alibaba/sealer/test/testhelper/settings"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func Login() {
	// check if docker json already exist
	config := DefaultRegistryAuthConfigDir()
	if utils.IsFileExist(config) {
		return
	}
	sess, err := testhelper.Start(fmt.Sprintf("%s login %s -u %s -p %s", settings.DefaultSealerBin, settings.RegistryURL,
		settings.RegistryUsername,
		settings.RegistryPasswd))

	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(sess).Should(gbytes.Say(fmt.Sprintf("login %s success", settings.RegistryURL)))
	gomega.Eventually(sess, settings.MaxWaiteTime).Should(gexec.Exit(0))
}

func Logout() {
	err := CleanLoginFile()
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func CleanLoginFile() error {
	return os.RemoveAll(DefaultRegistryAuthConfigDir())
}
func DefaultRegistryAuthConfigDir() string {
	dir, err := homedir.Dir()
	if err != nil {
		return settings.DefaultRegistryAuthFile
	}

	return filepath.Join(dir, ".docker/config.json")
}
