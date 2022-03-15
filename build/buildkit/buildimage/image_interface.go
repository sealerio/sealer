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

package buildimage

import (
	"github.com/alibaba/sealer/build/buildkit/buildinstruction"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type Executor interface {
	// Execute all raw layers,and merge with base layers.
	Execute(ctx Context, rawLayers []v1.Layer) ([]v1.Layer, error)
	Cleanup() error
}

type Differ interface {
	// Process :diff changes by build-in handler and save to dst,like pull docker image from manifests or helm charts
	//diff Metadata file changes save to the base layer.generally dst is the rootfs.
	Process(src, dst buildinstruction.MountTarget) error
}

type ImageSetter interface {
	// Set :fill up v1.image struct, like image annotations, platform and so on.
	Set(*v1.Image) error
}

type Middleware interface {
	// Process set data to cloud image ,but not to show in the image layer.
	Process(context, rootfs string) error
}

type ImageSaver interface {
	// Save with image attribute,and register to image metadata.
	Save(image *v1.Image) error
}
