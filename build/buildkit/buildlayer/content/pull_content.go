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

package content

import (
	"context"
	"github.com/alibaba/sealer/image/save"
	"github.com/alibaba/sealer/logger"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
)

type pullContent struct {
	puller  save.ImageSave
	ctx     context.Context
	saveDir string
}

func (p pullContent) Pull(images []string, platform ocispecs.Platform) error {
	err := p.puller.SaveImages(images, p.saveDir, platform)
	if err != nil {
		logger.Error("failed to pull cache image with error :%v", err)
		return err
	}
	return nil
}

func NewPullContent(saveDir string) Processor {
	//new registry puller
	ctx := context.Background()
	p := save.NewImageSaver(ctx)
	return pullContent{
		puller:  p,
		ctx:     ctx,
		saveDir: saveDir,
	}
}
