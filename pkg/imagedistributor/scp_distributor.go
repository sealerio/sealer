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
	"net"
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/pkg/config"
	"github.com/sealerio/sealer/pkg/env"
	"github.com/sealerio/sealer/pkg/infradriver"
	v1 "github.com/sealerio/sealer/types/api/v1"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	RegistryDirName      = "registry"
	RootfsCacheDirName   = "image"
	RegistryCacheDirName = "registry"
)

type scpDistributor struct {
	configs          []v1.Config
	infraDriver      infradriver.InfraDriver
	imageMountInfo   []ClusterImageMountInfo
	registryCacheDir string
	rootfsCacheDir   string
	options          DistributeOption
}

func (s *scpDistributor) DistributeRegistry(deployHosts []net.IP, dataDir string) error {
	for _, info := range s.imageMountInfo {
		if !osi.IsFileExist(filepath.Join(info.MountDir, RegistryDirName)) {
			continue
		}

		localCacheFile := filepath.Join(info.MountDir, info.ImageID)
		remoteCacheFile := filepath.Join(s.registryCacheDir, info.ImageID)
		eg, _ := errgroup.WithContext(context.Background())

		for _, deployHost := range deployHosts {
			tmpDeployHost := deployHost
			eg.Go(func() error {
				if !s.options.IgnoreCache {
					// detect if remote cache file is exist.
					existed, err := s.infraDriver.IsFileExist(tmpDeployHost, remoteCacheFile)
					if err != nil {
						return fmt.Errorf("failed to detect registry cache %s on host %s: %v",
							remoteCacheFile, tmpDeployHost.String(), err)
					}

					if existed {
						logrus.Debugf("cache %s hits on: %s, skip to do distribution", info.ImageID, tmpDeployHost.String())
						return nil
					}
				}

				// copy registry data
				err := s.infraDriver.Copy(tmpDeployHost, filepath.Join(info.MountDir, RegistryDirName), dataDir)
				if err != nil {
					return fmt.Errorf("failed to copy registry data %s: %v", info.MountDir, err)
				}

				// write cache flag
				err = s.writeCacheFlag(localCacheFile, remoteCacheFile, tmpDeployHost)
				if err != nil {
					return fmt.Errorf("failed to write registry cache %s on host %s: %v",
						remoteCacheFile, tmpDeployHost.String(), err)
				}

				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return err
		}
	}

	return nil
}

func (s *scpDistributor) Distribute(hosts []net.IP, dest string) error {
	for _, info := range s.imageMountInfo {
		if err := s.dumpConfigToRootfs(info.MountDir); err != nil {
			return err
		}

		if err := s.renderRootfs(info.MountDir); err != nil {
			return err
		}

		eg, _ := errgroup.WithContext(context.Background())
		localCacheFile := filepath.Join(info.MountDir, info.ImageID)
		remoteCacheFile := filepath.Join(s.rootfsCacheDir, info.ImageID)

		for _, ip := range info.Hosts {
			host := ip
			eg.Go(func() error {
				if !s.options.IgnoreCache {
					// detect if remote cache file is exist.
					existed, err := s.infraDriver.IsFileExist(host, remoteCacheFile)
					if err != nil {
						return fmt.Errorf("failed to detect rootfs cache %s on host %s: %v",
							remoteCacheFile, host.String(), err)
					}

					if existed {
						logrus.Debugf("cache %s hits on: %s, skip to do distribution", info.ImageID, host.String())
						return nil
					}
				}

				// copy rootfs data
				err := s.filterCopy(info.MountDir, dest, host)
				if err != nil {
					return fmt.Errorf("failed to copy rootfs files: %v", err)
				}

				// write cache flag
				err = s.writeCacheFlag(localCacheFile, remoteCacheFile, host)
				if err != nil {
					return fmt.Errorf("failed to write rootfs cache %s on host %s: %v",
						remoteCacheFile, host.String(), err)
				}

				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			return err
		}
	}

	return nil
}

func (s *scpDistributor) filterCopy(mountDir, dest string, host net.IP) error {
	files, err := os.ReadDir(mountDir)
	if err != nil {
		return fmt.Errorf("failed to read dir %s: %s", mountDir, err)
	}

	for _, f := range files {
		//skip registry directory
		if f.IsDir() && f.Name() == RegistryDirName {
			continue
		}

		// copy rootfs data
		err = s.infraDriver.Copy(host, filepath.Join(mountDir, f.Name()), filepath.Join(dest, f.Name()))
		if err != nil {
			return fmt.Errorf("failed to copy rootfs files: %v", err)
		}
	}

	return nil
}

func (s *scpDistributor) dumpConfigToRootfs(mountDir string) error {
	return config.NewConfiguration(mountDir).Dump(s.configs)
}

// using cluster render data to render Rootfs files
func (s *scpDistributor) renderRootfs(mountDir string) error {
	var (
		renderEtc       = filepath.Join(mountDir, "etc")
		renderChart     = filepath.Join(mountDir, "charts")
		renderManifests = filepath.Join(mountDir, "manifests")
		renderData      = s.infraDriver.GetClusterEnv()
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

// writeCacheFlag : write image sha256ID to remote host.
// remoteCacheFile looks like: /var/lib/sealer/data/my-cluster/rootfs/cache/registry/9eb6f8a1ca09559189dd1fed5e587b14
func (s *scpDistributor) writeCacheFlag(localCacheFile, remoteCacheFile string, host net.IP) error {
	if !osi.IsFileExist(localCacheFile) {
		err := osi.NewCommonWriter(localCacheFile).WriteFile([]byte(""))
		if err != nil {
			return fmt.Errorf("failed to write local cache file %s: %v", localCacheFile, err)
		}
	}

	err := s.infraDriver.Copy(host, localCacheFile, remoteCacheFile)
	if err != nil {
		return fmt.Errorf("failed to copy rootfs cache file: %v", err)
	}

	logrus.Debugf("successfully write cache file %s on: %s", remoteCacheFile, host.String())
	return nil
}

func (s *scpDistributor) Restore(targetDir string, hosts []net.IP) error {
	if !s.options.Prune {
		return nil
	}

	rmRootfsCMD := fmt.Sprintf("rm -rf %s", targetDir)

	eg, _ := errgroup.WithContext(context.Background())
	for _, ip := range hosts {
		host := ip
		eg.Go(func() error {
			err := s.infraDriver.CmdAsync(host, nil, rmRootfsCMD)
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

func NewScpDistributor(imageMountInfo []ClusterImageMountInfo, driver infradriver.InfraDriver, configs []v1.Config, options DistributeOption) (Distributor, error) {
	return &scpDistributor{
		configs:          configs,
		imageMountInfo:   imageMountInfo,
		infraDriver:      driver,
		registryCacheDir: filepath.Join(driver.GetClusterRootfsPath(), "cache", RegistryCacheDirName),
		rootfsCacheDir:   filepath.Join(driver.GetClusterRootfsPath(), "cache", RootfsCacheDirName),
		options:          options,
	}, nil
}
