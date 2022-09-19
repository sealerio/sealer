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

package imagedistributor

import (
	"context"
	"fmt"
	"github.com/sealerio/sealer/pkg/config"
	"github.com/sealerio/sealer/pkg/env"
	osi "github.com/sealerio/sealer/utils/os"
	"io/ioutil"

	"net"
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/common"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	v1 "github.com/sealerio/sealer/types/api/v1"

	"golang.org/x/sync/errgroup"
)

const (
	RegistryDirName = "registry"
)

type scpDistributor struct {
	configs     []v1.Config
	infraDriver infradriver.InfraDriver
	imageEngine imageengine.Interface
}

func (s *scpDistributor) Distribute(imageName string, hosts []net.IP) error {
	var (
		rootfs = s.infraDriver.GetClusterRootfs()
	)

	hostsPlatformMap, err := s.infraDriver.GetHostsPlatform(hosts)
	if err != nil {
		return err
	}

	for platform, targetHosts := range hostsPlatformMap {
		if err = s.pull(imageName, platform); err != nil {
			return err
		}

		mountDir, err := s.buildRootfs(imageName)
		if err != nil {
			return err
		}

		if err = s.dumpConfigToRootfs(mountDir); err != nil {
			return err
		}

		if err = s.renderRootfs(mountDir); err != nil {
			return err
		}

		targetDirs, err := s.filterRootfs(mountDir)
		if err != nil {
			return err
		}

		for _, target := range targetDirs {
			err = s.copyRootfs(target, filepath.Join(rootfs, filepath.Base(target)), targetHosts)
			if err != nil {
				return err
			}
		}

		if err = s.cleanRootfs(mountDir); err != nil {
			return fmt.Errorf("failed to remove mounted dir %s: %v", mountDir, err)
		}
	}

	return nil
}

func (s *scpDistributor) pull(imageName string, plat v1.Platform) error {
	// pull cluster image via it`s platform
	return s.imageEngine.Pull(&imagecommon.PullOptions{
		Quiet:      false,
		TLSVerify:  true,
		PullPolicy: "missing",
		Image:      imageName,
		Platform:   plat.ToString(),
	})
}

func (s *scpDistributor) buildRootfs(imageName string) (string, error) {
	mountDir := filepath.Join(common.DefaultSealerDataDir, "mount")

	if _, err := s.imageEngine.BuildRootfs(&imagecommon.BuildRootfsOptions{
		DestDir:       mountDir,
		ImageNameOrID: imageName,
	}); err != nil {
		return "", err
	}

	return mountDir, nil
}

func (s *scpDistributor) filterRootfs(mountDir string) ([]string, error) {
	var AllMountFiles []string

	files, err := ioutil.ReadDir(mountDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir %s: %s", mountDir, err)
	}

	for _, f := range files {
		//skip registry directory
		if f.IsDir() && f.Name() == RegistryDirName {
			continue
		}
		AllMountFiles = append(AllMountFiles, filepath.Join(mountDir, f.Name()))
	}

	return AllMountFiles, nil
}

func (s *scpDistributor) cleanRootfs(mountDir string) error {
	// umount cluster image this will remove all buildah containers
	if err := s.imageEngine.RemoveContainer(&imagecommon.RemoveContainerOptions{
		ContainerNamesOrIDs: nil,
		All:                 true,
	}); err != nil {
		return fmt.Errorf("remove containers failed, you'd better execute a prune to remove it: %v", err)
	}

	if err := os.RemoveAll(mountDir); err != nil {
		return err
	}

	return nil
}

func (s *scpDistributor) copyRootfs(mountDir, targetDir string, hosts []net.IP) error {
	eg, _ := errgroup.WithContext(context.Background())

	for _, ip := range hosts {
		host := ip
		eg.Go(func() error {
			err := s.infraDriver.Copy(host, mountDir, targetDir)
			if err != nil {
				return fmt.Errorf("failed to copy rootfs files: %v", err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (s *scpDistributor) dumpConfigToRootfs(mountDir string) error {
	return config.NewConfiguration(mountDir).Dump(s.configs)
}

// using cluster render data to render Rootfs files
func (s *scpDistributor) renderRootfs(mountDir string) error {
	var (
		renderEtc       = filepath.Join(mountDir, common.EtcDir)
		renderChart     = filepath.Join(mountDir, common.RenderChartsDir)
		renderManifests = filepath.Join(mountDir, common.RenderManifestsDir)
		renderData      = s.infraDriver.GetClusterRenderData()
	)

	for _, dir := range []string{renderEtc, renderChart, renderManifests} {
		if osi.IsFileExist(dir) {
			err := env.RenderTemplate(dir, renderData)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *scpDistributor) Restore(targetDir string, hosts []net.IP) error {
	rmRootfsCMD := fmt.Sprintf("rm -rf %s", targetDir)

	eg, _ := errgroup.WithContext(context.Background())
	for _, ip := range hosts {
		host := ip
		eg.Go(func() error {
			err := s.infraDriver.CmdAsync(host, rmRootfsCMD)
			if err != nil {
				return fmt.Errorf("faild to delete rootfs on host [%s]: %v", host.String(), err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func NewScpDistributor(imageEngine imageengine.Interface, driver infradriver.InfraDriver, configs []v1.Config) (Interface, error) {
	return &scpDistributor{
		configs:     configs,
		imageEngine: imageEngine,
		infraDriver: driver}, nil
}
