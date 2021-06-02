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

// +build darwin

package mount

type Interface interface {
	// return mount target merged dir, if target is "", using default dir name : [dir hash]/merged
	Mount(target string, upperDir string, layers ...string) error
	Unmount(target string) error
}

func NewMountDriver() Interface {
	return &Default{}
}
