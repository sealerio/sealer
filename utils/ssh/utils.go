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

package ssh

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
	dockerstreams "github.com/docker/cli/cli/streams"
	dockerioutils "github.com/docker/docker/pkg/ioutils"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/streamformatter"
)

func displayInit() {
	reader, writer = io.Pipe()
	writeFlusher = dockerioutils.NewWriteFlusher(writer)
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
		_ = writeFlusher.Close()
	}()
	progressChanOut = streamformatter.NewJSONProgressOutput(writeFlusher, false)
	err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, dockerstreams.NewOut(common.StdOut), nil)
	if err != nil && err != io.ErrClosedPipe {
		logger.Warn("error occurs in display progressing, err: %s", err)
	}
}

func localMd5Sum(localPath string) string {
	md5, err := utils.FileMD5(localPath)
	if err != nil {
		logger.Error("get file md5 failed %v", err)
		return ""
	}
	return md5
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func ReadPipe(stdout, stderr io.Reader, isStdout bool) {
	var combineSlice []string
	var combineLock sync.Mutex
	doneout := make(chan error, 1)
	doneerr := make(chan error, 1)
	go func() {
		doneerr <- readPipe(stderr, &combineSlice, &combineLock, isStdout)
	}()
	go func() {
		doneout <- readPipe(stdout, &combineSlice, &combineLock, isStdout)
	}()
	<-doneerr
	<-doneout
}

func readPipe(pipe io.Reader, combineSlice *[]string, combineLock *sync.Mutex, isStdout bool) error {
	r := bufio.NewReader(pipe)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return err
		}

		combineLock.Lock()
		*combineSlice = append(*combineSlice, string(line))
		if isStdout {
			fmt.Println(string(line))
		}
		combineLock.Unlock()
	}
}

func GetClusterPlatform(cluster *v2.Cluster) (map[string]v1.Platform, error) {
	clusterStatus := make(map[string]v1.Platform)
	for _, ip := range append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...) {
		IP := ip
		ssh, err := GetHostSSHClient(IP, cluster)
		if err != nil {
			return nil, err
		}
		clusterStatus[IP], err = ssh.Platform(IP)
		if err != nil {
			return nil, err
		}
	}
	return clusterStatus, nil
}

func WaitSSHReady(ssh Interface, tryTimes int, hosts ...string) error {
	var err error
	eg, _ := errgroup.WithContext(context.Background())
	for _, h := range hosts {
		host := h
		eg.Go(func() error {
			for i := 0; i < tryTimes; i++ {
				err = ssh.Ping(host)
				if err == nil {
					return nil
				}
				time.Sleep(time.Duration(i) * time.Second)
			}
			return fmt.Errorf("wait for [%s] ssh ready timeout:  %v, ensure that the IP address or password is correct", host, err)
		})
	}
	return eg.Wait()
}
