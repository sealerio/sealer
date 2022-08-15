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
	"fmt"

	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/containers/common/libimage"
	"github.com/pkg/errors"
)

func (engine *Engine) Tag(opts *options.TagOptions) error {
	name := opts.ImageNameOrID
	if len(name) == 0 {
		return errors.New("at least the image name or id should be specified")
	}
	if len(opts.Tags) == 0 {
		return errors.New("at least one new tag should be provided")
	}

	lookupOptions := &libimage.LookupImageOptions{ManifestList: true}
	existImage, _, err := engine.ImageRuntime().LookupImage(name, lookupOptions)
	if err != nil {
		return fmt.Errorf("failed to lookup image: %v", err)
	}

	for _, tag := range opts.Tags {
		if err := existImage.Tag(tag); err != nil {
			return err
		}
	}

	return nil
}
