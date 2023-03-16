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
	"strings"

	"github.com/sealerio/sealer/test/testhelper/settings"
	"github.com/sealerio/sealer/utils/exec"
)

// GetBuildImageName return specific image name for sealer build test
func GetBuildImageName() string {
	buildImageName := "docker.io/sealerio/build-test:v1"
	if settings.RegistryURL != "" && settings.RegistryUsername != "" && settings.RegistryPasswd != "" {
		buildImageName = settings.RegistryURL + "/" + settings.RegistryUsername + "/" + "build-test:v1"
	}
	return buildImageName
}

func WithCmdsBuildDir() string {
	return filepath.Join(settings.DefaultTestEnvDir, "suites", "build", "fixtures",
		"build_with_cmds")
}

func WithImageListFlagBuildDir() string {
	return filepath.Join(settings.DefaultTestEnvDir, "suites", "build", "fixtures",
		"build_with_imagelist_flag")
}

func WithLaunchBuildDir() string {
	return filepath.Join(settings.DefaultTestEnvDir, "suites", "build", "fixtures",
		"build_with_launch")
}

func WithAPPCmdsBuildDir() string {
	return filepath.Join(settings.DefaultTestEnvDir, "suites", "build", "fixtures",
		"build_with_appcmds")
}

func WithMultiArchBuildDir() string {
	return filepath.Join(settings.DefaultTestEnvDir, "suites", "build", "fixtures",
		"build_with_multi_arch")
}

type ArgsOfBuild struct {
	KubeFile, ImageName, Context string
	Platform                     []string
	ImageList                    string
	ImageType                    string
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

func (a *ArgsOfBuild) SetPlatforms(platforms []string) *ArgsOfBuild {
	a.Platform = platforms
	return a
}

func (a *ArgsOfBuild) SetImageList(imageList string) *ArgsOfBuild {
	a.ImageList = imageList
	return a
}

func (a *ArgsOfBuild) SetImageType(imageType string) *ArgsOfBuild {
	a.ImageType = imageType
	return a
}

func (a *ArgsOfBuild) String() string {
	if settings.DefaultSealerBin == "" || a.KubeFile == "" || a.ImageName == "" {
		return ""
	}

	var buildFlags []string
	buildFlags = append(buildFlags, fmt.Sprintf("%s build", settings.DefaultSealerBin))

	// add kubefile flag
	if a.KubeFile != "" {
		buildFlags = append(buildFlags, fmt.Sprintf("-f %s", a.KubeFile))
	}

	// add image tag flag
	if a.ImageName != "" {
		buildFlags = append(buildFlags, fmt.Sprintf("-t %s", a.ImageName))
	}

	// add image list flag
	if a.ImageList != "" {
		buildFlags = append(buildFlags, fmt.Sprintf("--image-list %s", a.ImageList))
	}

	// add platform flag
	if len(a.Platform) != 0 {
		buildFlags = append(buildFlags, fmt.Sprintf("--platform %s", strings.Join(a.Platform, ",")))
	}

	// add image type
	if a.ImageType != "" {
		buildFlags = append(buildFlags, fmt.Sprintf("--type %s", a.ImageType))
	}

	// add build context
	if a.Context == "" {
		a.Context = "."
	}
	buildFlags = append(buildFlags, a.Context)

	return strings.Join(buildFlags, " ")
}

func NewArgsOfBuild() *ArgsOfBuild {
	return &ArgsOfBuild{}
}

func CheckIsMultiArchImageExist(imageName string) bool {
	cmd := fmt.Sprintf("%s alpha manifest inspect %s", settings.DefaultSealerBin, imageName)
	_, err := exec.RunSimpleCmd(cmd)
	return err == nil
}

func CheckIsImageExist(imageName string) bool {
	cmd := fmt.Sprintf("%s inspect %s", settings.DefaultSealerBin, imageName)
	_, err := exec.RunSimpleCmd(cmd)
	return err == nil
}
