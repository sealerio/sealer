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

package version

// VersionedApplication TODO maybe move it a global version interface, for version compatibility
type VersionedApplication interface {
	Version() string

	Name() string

	Type() string

	Files() []string

	SetEnv(appEnv map[string]string)

	SetCmds(appCmds []string)
}
