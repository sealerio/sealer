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

package applyentity

import (
	"fmt"

	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/guest"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type ScaleApply struct {
	FileSystem      filesystem.Interface
	Runtime         runtime.Interface
	Guest           guest.Interface
	MastersToJoin   []string
	MastersToDelete []string
	NodesToJoin     []string
	NodesToDelete   []string
	IsScaleUp       bool
}

// DoApply do apply: do truly apply,input is desired cluster .
func (s ScaleApply) DoApply(cluster *v1.Cluster) error {
	/*
		1. master scale up + master scale up :support
		2. master scale down + master scale down :support
		3. master scale up + node scale down: not support
		4. master scale up + master scale down: not support
	*/
	runTime, err := runtime.NewDefaultRuntime(cluster)
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}
	s.Runtime = runTime

	if s.IsScaleUp {
		return s.ScaleUp(cluster)
	}
	return s.ScaleDown()
}

func (s ScaleApply) ScaleUp(cluster *v1.Cluster) error {
	hosts := append(s.MastersToJoin, s.NodesToJoin...)
	err := s.FileSystem.MountRootfs(cluster, hosts, true)
	if err != nil {
		return err
	}
	err = s.Runtime.JoinMasters(s.MastersToJoin)
	if err != nil {
		return err
	}
	err = s.Runtime.JoinNodes(s.NodesToJoin)
	if err != nil {
		return err
	}
	err = s.Guest.Apply(cluster)
	if err != nil {
		return err
	}
	return nil
}

func (s ScaleApply) ScaleDown() error {
	err := s.Runtime.DeleteMasters(s.MastersToDelete)
	if err != nil {
		return err
	}
	err = s.Runtime.DeleteNodes(s.NodesToDelete)
	if err != nil {
		return err
	}
	return nil
}

func NewScaleApply(fs filesystem.Interface, masterToDelete, masterToJoin, nodeToDelete, nodeToJoin []string) (Interface, error) {
	var up bool
	gs, err := guest.NewGuestManager()
	if err != nil {
		return nil, err
	}

	// only scale up or scale down at a time
	if len(masterToJoin) > 0 || len(nodeToJoin) > 0 {
		up = true
	}

	return ScaleApply{
		MastersToDelete: masterToDelete,
		MastersToJoin:   masterToJoin,
		NodesToDelete:   nodeToDelete,
		NodesToJoin:     nodeToJoin,
		IsScaleUp:       up,
		FileSystem:      fs,
		Guest:           gs,
	}, nil
}
