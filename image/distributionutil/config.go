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

package distributionutil

import (
	"time"

	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	"github.com/docker/docker/pkg/progress"
)

type Config struct {
	LayerStore     store.LayerStore
	ProgressOutput progress.Output
	Named          reference.Named
}

type registryConfig struct {
	Domain   string
	Insecure bool
	SkipPing bool
	NonSSL   bool
	Timeout  time.Duration
	Headers  map[string]string
}
