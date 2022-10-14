// Copyright © 2022 Alibaba Group Holding Ltd.
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

package buildah

import (
	"path/filepath"
	"runtime"

	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/image/v5/types"

	"github.com/sealerio/sealer/pkg/auth"

	"github.com/BurntSushi/toml"
	"github.com/containers/common/libimage"
	"github.com/containers/storage"
	"github.com/containers/storage/drivers/overlay"
	types2 "github.com/containers/storage/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/sealerio/sealer/pkg/define/options"
)

type Engine struct {
	*cobra.Command
	libimageRuntime *libimage.Runtime
	imageStore      storage.Store
}

func (engine *Engine) ImageRuntime() *libimage.Runtime {
	return engine.libimageRuntime
}

func (engine *Engine) ImageStore() storage.Store {
	return engine.imageStore
}

func (engine *Engine) SystemContext() *types.SystemContext {
	return engine.libimageRuntime.SystemContext()
}

func checkOverlaySupported() {
	conf := types2.TomlConfig{}
	if _, err := toml.DecodeFile(storageConfPath, &conf); err != nil {
		logrus.Warnf("failed to decode storage.conf, which may incur problems: %v", err)
		return
	}

	if conf.Storage.RunRoot == "" || conf.Storage.GraphRoot == "" {
		logrus.Warnf("runroot or graphroot is empty")
		return
	}

	// this check aims to register "overlay" and "overlay2" driver.
	// Otherwise, there will be "overlay" unsupported problem.
	// This issue is relevant with low-level library problem.
	// This is a weird problem. So fix it in this way currently.
	if _, err := overlay.SupportsNativeOverlay(
		filepath.Join(conf.Storage.GraphRoot, "overlay"),
		filepath.Join(conf.Storage.RunRoot, "overlay")); err != nil {
		logrus.Warnf("detect there is no native overlay supported: %v", err)
	}
}

// TODO we can provide a configuration file to export those options.
// the detailed information in the parse.SystemContextFromOptions
func systemContext() *types.SystemContext {
	// TODO
	// options for the following
	// DockerCertPath
	// tls-verify
	// os
	// arch
	// variant
	return &types.SystemContext{
		DockerRegistryUserAgent:           "Buildah/1.25.0",
		AuthFilePath:                      auth.GetDefaultAuthFilePath(),
		BigFilesTemporaryDir:              parse.GetTempDir(),
		OSChoice:                          runtime.GOOS,
		ArchitectureChoice:                runtime.GOARCH,
		DockerInsecureSkipTLSVerify:       types.NewOptionalBool(false),
		OCIInsecureSkipTLSVerify:          false,
		DockerDaemonInsecureSkipTLSVerify: false,
	}
}

func NewBuildahImageEngine(configurations options.EngineGlobalConfigurations) (*Engine, error) {
	if err := initBuildah(); err != nil {
		return nil, err
	}

	checkOverlaySupported()

	store, err := getStore(&configurations)
	if err != nil {
		return nil, err
	}

	sysCxt := systemContext()
	imageRuntime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: sysCxt})
	if err != nil {
		return nil, err
	}

	return &Engine{
		Command:         &cobra.Command{},
		libimageRuntime: imageRuntime,
		imageStore:      store,
	}, nil
}
