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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"sync"
	"syscall"
)

const (
	EtcdGitHubOrg    = "etcd-io"
	EtcdGithubRepo   = "etcd"
	GOOSLinux        = "linux"
	EtcdArtifactType = "etcd"
	EtcdVersion      = "v3.4.24"
	EtcdDestination  = "/tmp/etcd.tar.gz"
	EtcdBinName      = "etcd"
	EtcdctlBinName   = "etcdctl"
	WeedDestination  = "/tmp/weed.tar.gz"
	WeedBinName      = "weed"
)

func etcdDownloadURL() (string, error) {
	var ext string

	switch runtime.GOOS {
	case GOOSLinux:
		ext = ".tar.gz"
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// For the function stability, we use the specific version of etcd.
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s-%s-%s-%s%s",
		EtcdGitHubOrg, EtcdGithubRepo, EtcdVersion, EtcdArtifactType, EtcdVersion, runtime.GOOS, runtime.GOARCH, ext)

	return downloadURL, nil
}

type etcd struct {
	dataDir    string
	logDir     string
	pidDir     string
	binDir     string
	clientURL  string
	peerURL    string
	peers      []string
	wg         *sync.WaitGroup
	configFile string
}

// Etcd is the interface for etcd cluster.
type Etcd interface {
	Exec
}

type DeleteOptions struct {
	RetainLogs bool
}

type RunOptions struct {
	Binary string
	Name   string

	pidDir string
	logDir string
	args   []string
}

func NewEtcd(config *Config) Etcd {
	return &etcd{
		dataDir:    config.DataDir,
		logDir:     config.LogDir,
		pidDir:     config.PidDir,
		binDir:     config.BinDir,
		peers:      config.MasterIP,
		peerURL:    config.CurrentIP + ":" + strconv.Itoa(config.PeerPort),
		clientURL:  config.CurrentIP + ":" + strconv.Itoa(config.ClientPort),
		wg:         new(sync.WaitGroup),
		configFile: config.EtcdConfigPath,
	}
}

func (e *etcd) Name() string {
	return "etcd"
}

func (e *etcd) Start(ctx context.Context, binary string) error {
	// Generate etcd config file.
	err := e.GenerateConfig()
	if err != nil {
		return err
	}

	option := &RunOptions{
		Binary: binary,
		Name:   e.Name(),
		logDir: e.logDir,
		pidDir: e.pidDir,
		args:   e.BuildArgs(ctx),
	}
	if err := runBinary(ctx, option, e.wg); err != nil {
		return err
	}

	return nil
}

func (e *etcd) BuildArgs(ctx context.Context, params ...interface{}) []string {
	return []string{
		"--config-file", e.configFile,
	}
}

// GenerateConfig creates etcd cluster config file.
func (e *etcd) GenerateConfig() error {
	initialCluster := ""
	index := 0
	for i, peer := range e.peers {
		if peer == e.peerURL {
			index = i
		}
		initialCluster += "node" + strconv.Itoa(i) + "=http://" + peer + ","
	}
	initialCluster = initialCluster[:len(initialCluster)-1]
	name := "node" + strconv.Itoa(index)
	configContent := fmt.Sprintf(`name: "%s"
data-dir: "%s"
initial-cluster-token: "my-etcd-token"
initial-cluster: "%s"
initial-advertise-peer-urls: "http://%s"
listen-peer-urls: "http://%s"
listen-client-urls: "http://%s"
advertise-client-urls: "http://%s"
log-file: "%s"
pid-file: "%s"
`, name, e.dataDir, initialCluster, e.peerURL, e.peerURL, e.clientURL, e.clientURL, e.logDir, e.pidDir)

	// write config file
	err := ioutil.WriteFile(e.configFile, []byte(configContent), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (e *etcd) IsRunning(ctx context.Context) bool {
	_, port, err := net.SplitHostPort(e.clientURL)
	if err != nil {
		return false
	}
	err = exec.Command("lsof", "-i:"+port).Run()
	return err == nil
}

func runBinary(ctx context.Context, option *RunOptions, wg *sync.WaitGroup) error {
	cmd := exec.CommandContext(ctx, option.Binary, option.args...)

	// output to binary.
	logFile := path.Join(option.logDir, "log")
	outputFile, err := os.Create(logFile)
	if err != nil {
		return err
	}

	outputFileWriter := bufio.NewWriter(outputFile)
	cmd.Stdout = outputFileWriter
	cmd.Stderr = outputFileWriter

	if err := cmd.Start(); err != nil {
		return err
	}

	pid := strconv.Itoa(cmd.Process.Pid)

	pidFile := path.Join(option.pidDir, "pid")
	f, err := os.Create(pidFile)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(pid))
	if err != nil {
		return err
	}

	go func() {
		defer wg.Done()
		wg.Add(1)
		if err := cmd.Wait(); err != nil {
			// Caught signal kill and interrupt error then ignore.
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				if status, ok := exit.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() &&
						(status.Signal() == syscall.SIGKILL || status.Signal() == syscall.SIGINT) {
						return
					}
				}
			}
			_ = outputFileWriter.Flush()
		}
	}()

	return nil
}

func runBinaryWithJSONResponse(ctx context.Context, option *RunOptions, wg *sync.WaitGroup) ([]byte, error) {
	cmd := exec.CommandContext(ctx, option.Binary, option.args...)

	var jsonOutput bytes.Buffer
	cmd.Stdout = &jsonOutput
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	//TODO if pid file == "", skip this step
	pid := strconv.Itoa(cmd.Process.Pid)

	pidFile := path.Join(option.pidDir, "pid")
	f, err := os.Create(pidFile)
	if err != nil {
		return nil, err
	}

	_, err = f.Write([]byte(pid))
	if err != nil {
		return nil, err
	}

	go func() {
		defer wg.Done()
		wg.Add(1)
		if err := cmd.Wait(); err != nil {
			// Caught signal kill and interrupt error then ignore.
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				if status, ok := exit.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() &&
						(status.Signal() == syscall.SIGKILL || status.Signal() == syscall.SIGINT) {
						return
					}
				}
			}
		}
	}()

	jsonResponse := jsonOutput.Bytes()
	return jsonResponse, nil
}

func CreateDirIfNotExists(dir string) (err error) {
	if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func IsFileExists(filepath string) (bool, error) {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		// file does not exist
		return false, nil
	}

	if err != nil {
		// Other errors happened.
		return false, err
	}

	if info.IsDir() {
		// It's a directory.
		return false, fmt.Errorf("'%s' is directory, not file", filepath)
	}

	// The file exists.
	return true, nil
}
