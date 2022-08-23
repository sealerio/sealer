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
	"errors"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/buildah"
	"github.com/containers/buildah/pkg/parse"
)

func (engine *Engine) GetImageAnnotation(opts *options.GetImageAnnoOptions) (map[string]string, error) {
	if len(opts.ImageNameOrID) == 0 {
		return nil, errors.New("image name id or image name should be specified")
	}

	var builder *buildah.Builder
	systemContext, err := parse.SystemContextFromOptions(engine.Command)
	if err != nil {
		return nil, err
	}

	ctx := getContext()
	store := engine.ImageStore()
	name := opts.ImageNameOrID

	builder, err = openImage(ctx, systemContext, store, name)
	if err != nil {
		return nil, err
	}

	out := buildah.GetBuildInfo(builder)
	return out.ImageAnnotations, nil
}
