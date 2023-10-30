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

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/sealerio/sealer/utils/exec"
)

type Volume interface {
	Exec
}

type volume struct {
	ip                string
	port              int
	dir               string
	mServer           []string
	needMoreLocalNode bool
	dirList           []string
	portList          []int
	wg                *sync.WaitGroup
}

func NewWeedVolume(config *Config, mServer []string) Volume {
	return &volume{
		ip:                config.CurrentIP,
		port:              config.WeedVolumePort,
		dir:               config.WeedVolumeDir,
		mServer:           mServer,
		needMoreLocalNode: config.NeedMoreLocalNode,
		wg:                new(sync.WaitGroup),
	}
}

func (v *volume) Start(ctx context.Context, binary string) error {
	if v.needMoreLocalNode {
		return v.startCluster(ctx, binary)
	}
	return v.startSingle(ctx, binary)
}

func (v *volume) BuildArgs(ctx context.Context, params ...interface{}) []string {
	return []string{
		"-mServer " + strings.Join(v.mServer, ","),
		"-port " + params[0].(string),
		"-dir " + params[1].(string),
	}
}

func (v *volume) IsRunning(ctx context.Context) bool {
	err := exec.Cmd("lsof", "-i:"+strconv.Itoa(v.port))
	return err == nil
}

func (v *volume) Name() string {
	return "volume"
}

func (v *volume) startCluster(ctx context.Context, binary string) error {
	for i := 0; i < len(v.portList); i++ {
		runOptions := &RunOptions{
			Binary: binary,
			Name:   "volume",
			args:   v.BuildArgs(ctx, strconv.Itoa(v.portList[i]), v.dirList[i]),
		}
		err := runBinary(ctx, runOptions, v.wg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *volume) startSingle(ctx context.Context, binary string) error {
	runOptions := &RunOptions{
		Binary: binary,
		Name:   "volume",
		args:   v.BuildArgs(ctx, strconv.Itoa(v.port), v.dir),
	}
	err := runBinary(ctx, runOptions, v.wg)
	if err != nil {
		return err
	}
	return nil
}
