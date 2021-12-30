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

package save

import (
	"context"

	"github.com/docker/docker/pkg/progress"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

//Image interface can save a list of images of the specified platform, it's not concurrent safe
type Image interface {
	SaveImages(images []string, platform v1.Platform) error
}

type chart struct {
	name string // e.g: mysql
	repo string // e.g: https://charts.bitnami.com/bitnami
}

// Chart interface can save a list of helm charts, it's not concurrent safe
type Chart interface {
	SaveCharts(charts []chart) error
}

//ChartImage interface can save both docker images and helm charts
type ChartImage interface {
	Image
	Chart
}

//DefaultSaver implement ChartImage interface
type DefaultSaver struct {
	ctx         context.Context
	rootdir     string
	progressOut progress.Output
}

//NewSaver receive two arguments and create a DefaultSaver
func NewSaver(ctx context.Context, rootdir string) ChartImage {
	if ctx == nil {
		ctx = context.Background()
	}
	if rootdir[len(rootdir)-1] == '/' {
		rootdir = rootdir[:len(rootdir)-1]
	}
	return &DefaultSaver{
		ctx:     ctx,
		rootdir: rootdir,
	}
}
