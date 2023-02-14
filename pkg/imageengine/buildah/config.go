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
	"strings"

	"github.com/containers/buildah"
	"github.com/pkg/errors"

	"github.com/sealerio/sealer/pkg/define/options"
)

func (engine *Engine) Config(opts *options.ConfigOptions) error {
	if len(opts.ContainerID) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	name := opts.ContainerID

	ctx := getContext()
	store := engine.ImageStore()
	builder, err := OpenBuilder(ctx, store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	if err := updateConfig(builder, opts); err != nil {
		return err
	}
	return builder.Save()
}

func updateConfig(builder *buildah.Builder, iopts *options.ConfigOptions) error {
	if len(iopts.Annotations) != 0 {
		for _, annotationSpec := range iopts.Annotations {
			annotation := strings.SplitN(annotationSpec, "=", 2)
			switch {
			case len(annotation) > 1:
				builder.SetAnnotation(annotation[0], annotation[1])
			case annotation[0] == "-":
				builder.ClearAnnotations()
			case strings.HasSuffix(annotation[0], "-"):
				builder.UnsetAnnotation(strings.TrimSuffix(annotation[0], "-"))
			default:
				builder.SetAnnotation(annotation[0], "")
			}
		}
	}
	return nil
}
