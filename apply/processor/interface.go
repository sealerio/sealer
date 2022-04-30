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

package processor

import (
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type Interface interface {
	// Execute :according to the different of desired cluster to do cluster apply.
	Execute(cluster *v2.Cluster) error
}

type Processor interface {
	GetPipeLine() ([]func(cluster *v2.Cluster) error, error)
}

type Executor struct {
	Processor
}

func NewExecutor(proc Processor) Interface {
	return &Executor{proc}
}

func (e *Executor) Execute(cluster *v2.Cluster) error {
	pipLine, err := e.GetPipeLine()
	if err != nil {
		return err
	}

	for _, f := range pipLine {
		if err = f(cluster); err != nil {
			return err
		}
	}
	return nil
}
