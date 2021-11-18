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

package checker

import (
	"fmt"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

const (
	PhasePre       = "Pre"
	PhasePost      = "Post"
	PhaseView      = "view"
	PhaseLiteBuild = "LiteBuild"
)

// Define checkers when pre or post install, like checker node status, checker pod status...
type Interface interface {
	Check(cluster *v1.Cluster, phase string) error
}

func RunViewCheckList(cluster *v1.Cluster) error {
	list := []Interface{NewNodeChecker(), NewSvcChecker(), NewPodChecker()}

	return RunCheckList(list, cluster, PhaseView)
}

func RunCheckList(list []Interface, cluster *v1.Cluster, phase string) error {
	for _, l := range list {
		if err := l.Check(cluster, phase); err != nil {
			return fmt.Errorf("failed to run checker: %v", err)
		}
	}
	return nil
}
