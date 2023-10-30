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

package weed

import "context"

// Exec is the interface for command execution.
// It provides the basic operations for command execution.
// Like start, build args, is running, name.
type Exec interface {
	// Start starts cluster component by executing binary.
	Start(ctx context.Context, binary string) error

	// BuildArgs build up args for cluster component.
	BuildArgs(ctx context.Context, params ...interface{}) []string

	// IsRunning returns the status of current cluster component.
	IsRunning(ctx context.Context) bool

	// Name return the name of component.
	Name() string
}
