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

package build

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alibaba/sealer/test/testhelper/settings"
	"github.com/alibaba/sealer/utils"
	"github.com/onsi/gomega"
)

func GetFixtures() string {
	return filepath.Join(settings.DefaultTestEnvDir, "suites", "build", "fixtures")
}

func GetLocalBuildDir() string {
	return "local_build"
}

func GetCloudBuildDir() string {
	return "cloud_build"
}

// GetTestImageName return specific image name that will be push to registry
func GetTestImageName() string {
	return fmt.Sprintf("sealer-io/%s%d:%s", settings.ImageName, 719, "v1")
}

//GetImageNameTemplate return image name only for local build
func GetImageNameTemplate(name string) string {
	return fmt.Sprintf("sealer-io-test/%s%s:%s", settings.ImageName, name, "v1")
}

type ArgsOfBuild struct {
	KubeFile, ImageName, Context, BuildType string
}

func (a *ArgsOfBuild) SetKubeFile(kubeFile string) *ArgsOfBuild {
	a.KubeFile = kubeFile
	return a
}

func (a *ArgsOfBuild) SetImageName(imageName string) *ArgsOfBuild {
	a.ImageName = imageName
	return a
}

func (a *ArgsOfBuild) SetContext(context string) *ArgsOfBuild {
	a.Context = context
	return a
}

func (a *ArgsOfBuild) SetBuildType(buildType string) *ArgsOfBuild {
	a.BuildType = buildType
	return a
}

func (a *ArgsOfBuild) Build() string {
	if settings.DefaultSealerBin == "" || a.KubeFile == "" || a.ImageName == "" {
		return ""
	}

	if a.Context == "" {
		a.Context = "."
	}

	if a.BuildType == "" {
		a.BuildType = settings.LocalBuild
	}
	return fmt.Sprintf("%s build -f %s -t %s -b %s %s", settings.DefaultSealerBin, a.KubeFile, a.ImageName, a.BuildType, a.Context)
}

func NewArgsOfBuild() *ArgsOfBuild {
	return &ArgsOfBuild{}
}

func CheckIsImageExist(imageName string) bool {
	cmd := fmt.Sprintf("%s images | grep %s | wc -l", settings.DefaultSealerBin, imageName)
	result, err := utils.RunSimpleCmd(cmd)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	num, err := strconv.Atoi(strings.Replace(result, "\n", "", -1))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return num == 1
}

func CheckClusterFile(imageName string) bool {
	cmd := fmt.Sprintf("%s images | grep %s | awk '{print $4}'", settings.DefaultSealerBin, imageName)
	result, err := utils.RunSimpleCmd(cmd)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	cmd = fmt.Sprintf("%s inspect -c %s | grep Cluster | wc -l", settings.DefaultSealerBin,
		strings.Replace(result, "\n", "", -1))
	result, err = utils.RunSimpleCmd(cmd)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	num, err := strconv.Atoi(strings.Replace(result, "\n", "", -1))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return num == 1
}
