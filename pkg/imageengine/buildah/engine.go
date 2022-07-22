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
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/spf13/cobra"
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

func NewBuildahImageEngine(configurations options.EngineGlobalConfigurations) (*Engine, error) {
	if err := initBuildah(); err != nil {
		return nil, err
	}

	store, err := getStore(&configurations)
	if err != nil {
		return nil, err
	}

	sysCxt := &types.SystemContext{BigFilesTemporaryDir: parse.GetTempDir()}
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
