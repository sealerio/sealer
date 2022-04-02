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

package buildinstruction

import (
	"github.com/opencontainers/go-digest"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/pkg/image/cache"
	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type ExecContext struct {
	BuildContext string
	BuildArgs    map[string]string
	//cache flag,will change for each layer ctx
	ContinueCache bool
	//cache chain to hit,will change for each layer ctx
	ParentID cache.ChainID
	//static method to do cache
	CacheSvc cache.Service
	Prober   image.Prober
	//used to gen layer
	LayerStore store.LayerStore
}

type InstructionContext struct {
	// dynamic method to init different instruction
	CurrentLayer *v1.Layer
	BaseLayers   []v1.Layer
	Platform     v1.Platform
}

type Out struct {
	LayerID       digest.Digest
	ParentID      cache.ChainID
	ContinueCache bool
}

func NewInstruction(ic InstructionContext) (Interface, error) {
	// init each inst via layer type
	switch ic.CurrentLayer.Type {
	case common.CMDCOMMAND, common.RUNCOMMAND:
		return NewCmdInstruction(ic)
	case common.COPYCOMMAND:
		return NewCopyInstruction(ic)
	}

	return nil, nil
}

func NewExecContext(buildContext string, buildArgs map[string]string, useCache bool, layerStore store.LayerStore) ExecContext {
	if !useCache {
		return ExecContext{
			LayerStore:   layerStore,
			BuildContext: buildContext,
			BuildArgs:    buildArgs,
		}
	}
	chainSvc, err := cache.NewService()
	if err != nil {
		return ExecContext{}
	}

	service, err := image.NewImageService()
	if err != nil {
		return ExecContext{}
	}

	prober := image.NewImageProber(service, true)
	return ExecContext{
		LayerStore:    layerStore,
		BuildContext:  buildContext,
		BuildArgs:     buildArgs,
		CacheSvc:      chainSvc,
		ParentID:      cache.ChainID(""),
		Prober:        prober,
		ContinueCache: true,
	}
}
