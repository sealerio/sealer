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
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/sealerio/sealer/utils/exec"
)

type Master interface {
	Exec
	UploadFile(ctx context.Context, master string, dir string) (UploadFileResponse, error)
	DownloadFile(ctx context.Context, master string, fid string, outputDir string) error
	RemoveFile(ctx context.Context, master string, dir string) error
}

type master struct {
	ip                 string
	port               int
	mDir               string
	defaultReplication string
	peers              []string
	needMoreLocalNode  bool
	portList           []int
	mDirList           []string
	wg                 *sync.WaitGroup
}

type UploadFileResponse struct {
	Fid      string `json:"fid"`
	URL      string `json:"url"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
}

func (m *master) UploadFile(ctx context.Context, master string, dir string) (UploadFileResponse, error) {
	runOptions := RunOptions{
		Binary: "weed",
		Name:   "upload",
		args:   m.buildUploadFileArgs(ctx, master, dir),
	}
	jsonResponse, err := runBinaryWithJSONResponse(ctx, &runOptions, m.wg)
	if err != nil {
		return UploadFileResponse{}, err
	}
	var uploadFileResponse UploadFileResponse
	err = json.Unmarshal(jsonResponse, &uploadFileResponse)
	if err != nil {
		return UploadFileResponse{}, err
	}
	return uploadFileResponse, nil
}

func (m *master) buildUploadFileArgs(ctx context.Context, params ...interface{}) []string {
	_ = ctx
	return []string{
		"-master=" + params[0].(string),
		"-dir=" + params[1].(string),
	}
}

func (m *master) buildDownloadFileArgs(ctx context.Context, params ...interface{}) []string {
	_ = ctx
	return []string{
		"-server=" + params[0].(string),
		"--dir=" + params[2].(string),
		params[1].(string),
	}
}

func (m *master) DownloadFile(ctx context.Context, master string, fid string, outputDir string) error {
	runOptions := RunOptions{
		Binary: "weed",
		Name:   "download",
		args:   m.buildDownloadFileArgs(ctx, master, fid, outputDir),
	}
	err := runBinary(ctx, &runOptions, m.wg)
	if err != nil {
		return err
	}
	return nil
}

func (m *master) RemoveFile(ctx context.Context, master string, fid string) error {
	//TODO weed may not support remove file, may be should consider to use other file system
	panic("implement me")
}

func (m *master) Start(ctx context.Context, binary string) error {
	if m.needMoreLocalNode {
		return m.startCluster(ctx, binary)
	}
	return m.startSingle(ctx, binary)
}

func (m *master) BuildArgs(ctx context.Context, params ...interface{}) []string {
	return []string{
		"master",
		"-ip " + m.ip,
		"-port " + params[0].(string),
		"-mdir " + params[1].(string),
		"-peers " + strings.Join(m.peers, ","),
		"-defaultReplication " + m.defaultReplication,
	}
}

func (m *master) IsRunning(ctx context.Context) bool {
	err := exec.Cmd("lsof", "-i:"+strconv.Itoa(m.port))
	return err == nil
}

func (m *master) Name() string {
	return "master"
}

func NewMaster(config *Config) Master {
	return &master{
		ip:                 config.CurrentIP,
		port:               config.WeedMasterPort,
		mDir:               config.WeedMasterDir,
		defaultReplication: config.DefaultReplication,
		peers:              config.MasterIP,
		needMoreLocalNode:  config.NeedMoreLocalNode,
		wg:                 new(sync.WaitGroup),
	}
}

func (m *master) startSingle(ctx context.Context, binary string) error {
	runOptions := &RunOptions{
		Binary: binary,
		Name:   "master",
		args:   m.BuildArgs(ctx, strconv.Itoa(m.port), m.mDir),
	}
	err := runBinary(ctx, runOptions, m.wg)
	if err != nil {
		return err
	}
	return nil
}

func (m *master) startCluster(ctx context.Context, binary string) error {
	for i, port := range m.portList {
		runOptions := &RunOptions{
			Binary: binary,
			Name:   "master",
			args:   m.BuildArgs(ctx, strconv.Itoa(port), m.mDirList[i]),
		}
		err := runBinary(ctx, runOptions, m.wg)
		if err != nil {
			return err
		}
	}
	return nil
}
