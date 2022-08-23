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

	v1 "github.com/sealerio/sealer/types/api/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

func (engine *Engine) GetSealerImageExtension(opts *options.GetImageAnnoOptions) (v1.ImageExtension, error) {
	annotation, err := engine.GetImageAnnotation(opts)
	extension := v1.ImageExtension{}
	if err != nil {
		return extension, err
	}

	extensionStr := annotation[v1.SealerImageExtension]
	if len(extensionStr) == 0 {
		return extension, fmt.Errorf("%s does not exist in image %s", v1.SealerImageExtension, opts.ImageNameOrID)
	}

	err = json.Unmarshal([]byte(extensionStr), &extension)
	if err != nil {
		return extension, fmt.Errorf("failed to unmarshal %v for image %v: %v", v1.SealerImageExtension, opts.ImageNameOrID, err)
	}
	return extension, nil
}
