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

package v1

import (
	"encoding/json"

	v1 "github.com/sealerio/sealer/pkg/define/application/v1"
	"github.com/sealerio/sealer/pkg/define/application/version"
)

const (
	SealerImageExtension = "sealer.image.extension"
	// Kube image type
	KubeInstaller = "kube-installer"
	AppInstaller  = "app-installer"
)

type ImageExtension struct {
	// sealer image type, like AppImage
	Type string `json:"type,omitempty"`
	// applications in the sealer image
	Applications []version.VersionedApplication `json:"applications,omitempty"`
	// launch spec will declare
	Launch Launch `json:"launch,omitempty"`
}

type Launch struct {
	// user specified CMDS instruction
	Cmds []string `json:"cmds,omitempty"`
	// user specified LAUNCH instruction
	AppNames []string `json:"app_names,omitempty"`
}

type v1ImageExtension struct {
	// sealer image type, like AppImage
	Type string `json:"type,omitempty"`
	// applications in the sealer image
	Applications []v1.Application `json:"applications,omitempty"`
	// launch spec will declare
	Launch Launch `json:"launch,omitempty"`
}

func (ie *ImageExtension) UnmarshalJSON(data []byte) error {
	*ie = ImageExtension{}
	v1Ex := v1ImageExtension{}
	if err := json.Unmarshal(data, &v1Ex); err != nil {
		return err
	}

	(*ie).Type = v1Ex.Type
	(*ie).Applications = make([]version.VersionedApplication, len(v1Ex.Applications))
	for i, app := range v1Ex.Applications {
		tmpApp := app
		(*ie).Applications[i] = &tmpApp
	}
	(*ie).Launch = v1Ex.Launch
	return nil
}
