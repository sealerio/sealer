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

	image_v1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/define/options"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

func (engine *Engine) GetSealerImageExtension(opts *options.GetImageAnnoOptions) (image_v1.ImageExtension, error) {
	annotation, err := engine.GetImageAnnotation(opts)
	extension := image_v1.ImageExtension{}
	if err != nil {
		return extension, err
	}

	result, err := GetImageExtensionFromAnnotations(annotation)
	if err != nil {
		return extension, errors.Wrapf(err, "failed to get %s in image %s", image_v1.SealerImageExtension, opts.ImageNameOrID)
	}
	return result, nil
}

func GetImageExtensionFromAnnotations(annotations map[string]string) (image_v1.ImageExtension, error) {
	extension := image_v1.ImageExtension{}
	extensionStr := annotations[image_v1.SealerImageExtension]
	if len(extensionStr) == 0 {
		return extension, fmt.Errorf("%s does not exist", image_v1.SealerImageExtension)
	}

	if err := json.Unmarshal([]byte(extensionStr), &extension); err != nil {
		return extension, fmt.Errorf("failed to unmarshal %v: %v", image_v1.SealerImageExtension, err)
	}
	return extension, nil
}
