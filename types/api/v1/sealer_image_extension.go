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

const (
	SealerImageExtension = "sealer.image.extension"
)

type ImageExtension struct {
	// sealer image type, like AppImage
	ImageType string `json:"image_type"`
	// cmd list for launching
	CmdSet []string `json:"cmd_set"`
	// arg list for CmdSet
	ArgSet map[string]string `json:"arg_set"`
}
