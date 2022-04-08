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

package prune

const (
	LayerPruner = "start to prune layer"
	ImagePruner = "start to prune image db"
	BuildPruner = "start to prune build tmp"
)

type Pruner interface {
	// Select do select action and return filepath which need to be deleted
	Select() ([]string, error)
	GetSelectorMessage() string
}
