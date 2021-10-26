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

package buildkit

import (
	"fmt"
	"os"

	"github.com/alibaba/sealer/build/buildkit/buildinstruction"

	"github.com/alibaba/sealer/client/docker"
	"github.com/alibaba/sealer/runtime"
	"github.com/alibaba/sealer/utils/mount"

	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
)

const (
	kubefile = "Kubefile"
)

func NewRegistryCache() (*buildinstruction.MountTarget, error) {
	//$rootfs/registry
	dir := GetRegistryBindDir()
	if dir == "" {
		return nil, nil
	}
	rootfs := filepath.Dir(dir)
	isMounted, upper := mount.GetMountDetails(rootfs)
	if isMounted {
		logger.Info("get registry cache dir :%s success ", dir)
		return buildinstruction.NewMountTarget(rootfs, upper, []string{rootfs})
	}

	// if rootfs dir not mounted, unable to get cache image layer. need to mount rootfs before init-registry
	mountTarget, err := buildinstruction.NewMountTarget(rootfs, runtime.RegistryMountUpper, []string{rootfs})
	if err != nil {
		return nil, err
	}
	str, err := utils.RunSimpleCmd(fmt.Sprintf("rm -rf %s && mkdir -p %s", runtime.RegistryMountUpper, runtime.RegistryMountUpper))
	if err != nil {
		logger.Error(str)
		return nil, err
	}
	err = mountTarget.TempMount()
	if err != nil {
		return nil, fmt.Errorf("failed to mount %s, %v", rootfs, err)
	}
	str, err = utils.RunSimpleCmd(fmt.Sprintf("cd %s/scripts && sh init-registry.sh 5000 %s/registry", rootfs, rootfs))
	logger.Info(str)
	if err != nil {
		return nil, fmt.Errorf("failed to init registry, %s", err)
	}
	return mountTarget, nil
}

func GetRegistryBindDir() string {
	// check is docker running runtime.RegistryName
	// check bind dir
	var registryName = runtime.RegistryName
	var registryDest = runtime.RegistryBindDest

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return ""
	}

	containers, err := dockerClient.GetContainerListByName(registryName)

	if err != nil {
		return ""
	}

	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type == "bind" && m.Destination == registryDest {
				return m.Source
			}
		}
	}

	return ""
}

// ParseBuildArgs parse context and kubefile. return context abs path and kubefile abs path
func ParseBuildArgs(localContextDir, kubeFileName string) (string, string, error) {
	localDir, err := resolveAndValidateContextPath(localContextDir)
	if err != nil {
		return "", "", err
	}

	if kubeFileName != "" {
		if kubeFileName, err = filepath.Abs(kubeFileName); err != nil {
			return "", "", fmt.Errorf("unable to get absolute path to KubeFile: %v", err)
		}
	}

	relFileName, err := getKubeFileRelPath(localDir, kubeFileName)
	return localDir, relFileName, err
}

func resolveAndValidateContextPath(givenContextDir string) (string, error) {
	absContextDir, err := filepath.Abs(givenContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute context directory %s: %v", givenContextDir, err)
	}

	absContextDir, err = filepath.EvalSymlinks(absContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate symlinks in context path: %v", err)
	}

	stat, err := os.Lstat(absContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to stat context directory %s: %v", absContextDir, err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("context must be a directory: %s", absContextDir)
	}

	return absContextDir, err
}

func getKubeFileRelPath(absContextDir, givenKubeFile string) (string, error) {
	var err error

	absKubeFile := givenKubeFile
	if absKubeFile == "" {
		absKubeFile = filepath.Join(absContextDir, kubefile)
		if _, err = os.Lstat(absKubeFile); os.IsNotExist(err) {
			altPath := filepath.Join(absContextDir, strings.ToLower(kubefile))
			if _, err = os.Lstat(altPath); err == nil {
				absKubeFile = altPath
			}
		}
	}

	if !filepath.IsAbs(absKubeFile) {
		absKubeFile = filepath.Join(absContextDir, absKubeFile)
	}

	absKubeFile, err = filepath.EvalSymlinks(absKubeFile)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate symlinks in KubeFile path: %v", err)
	}

	if _, err := os.Lstat(absKubeFile); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("cannot locate KubeFile: %s", absKubeFile)
		}
		return "", fmt.Errorf("unable to stat KubeFile: %v", err)
	}

	return absKubeFile, nil
}

func ValidateContextDirectory(srcPath string) error {
	contextRoot, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}

	return filepath.Walk(contextRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("can't stat '%s'", filePath)
			}
			if os.IsNotExist(err) {
				return fmt.Errorf("file '%s' not found", filePath)
			}
			return err
		}

		if f.IsDir() {
			return nil
		}

		if f.Mode()&(os.ModeSymlink|os.ModeNamedPipe) != 0 {
			return nil
		}

		currentFile, err := os.Open(filePath)
		if err != nil && os.IsPermission(err) {
			return fmt.Errorf("no permission to read from '%s'", filePath)
		}
		currentFile.Close()

		return nil
	})
}
