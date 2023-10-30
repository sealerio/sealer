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
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"runtime"
)

// Config is the config of the weed cluster and etcd cluster.
type Config struct {
	// MasterIP is the IP address of the master node.
	MasterIP []string
	// VolumeIP is the IP address of the volume node.
	VolumeIP []string
	// LogDir is the directory of the log file.
	LogDir string
	// DataDir is the directory of the data file.
	DataDir string
	// PidDir is the directory of the pid file.
	PidDir string
	// BinDir is the directory of the etcd binary file.
	BinDir string
	// EtcdConfigPath is the path of the etcd config file, we will generate it automatically.
	EtcdConfigPath string
	// CurrentIP is the IP address of the current node.
	CurrentIP string
	// PeerPort is the port of the peer.
	PeerPort int
	// ClientPort is the port of the client.
	ClientPort int
	// WeedMasterPort is the port of the weed master node.
	WeedMasterPort int
	// WeedVolumePort is the port of the weed volume node.
	WeedVolumePort int
	// NeedMoreLocalNode is the flag of whether you need more local weed node.
	NeedMoreLocalNode bool
	// WeedMasterDir is the directory of the weed master node.
	WeedMasterDir string
	// WeedVolumeDir is the directory of the weed volume node.
	WeedVolumeDir string
	// DefaultReplication is the default replication of the weed cluster.
	DefaultReplication string
	// WeedLogDir is the directory of the weed log file.
	WeedLogDir string
	// weedMasterPortList is the port list of the weed master node when need more local weed node.
	weedMasterPortList []int
	// weedVolumePortList is the port list of the weed volume node when need more local weed node.
	weedVolumePortList []int
	// weedMDirList is the directory list of the weed master node when need more local weed node.
	weedMDirList []string
	// weedVDirList is the directory list of the weed volume node when need more local weed node.
	weedVDirList []string
	// weedMasterList is the list of the weed master node when need more local weed node.
	weedMasterList []string
}

type Deployer interface {
	// GetWeedMasterList returns the master list of the weed cluster.
	GetWeedMasterList(ctx context.Context) ([]string, error)

	// CreateEtcdCluster creates the etcd cluster.
	CreateEtcdCluster(ctx context.Context) error

	// DeleteEtcdCluster deletes the etcd cluster.
	DeleteEtcdCluster(ctx context.Context) error

	// CreateWeedCluster creates the weed cluster.
	CreateWeedCluster(ctx context.Context) error

	// DeleteWeedCluster deletes the weed cluster.
	DeleteWeedCluster(ctx context.Context) error

	// UploadFile uploads the file to the weed cluster.
	UploadFile(ctx context.Context, dir string) error

	// DownloadFile download the file from the weed cluster.
	DownloadFile(ctx context.Context, dir string, outputDir string) error

	// RemoveFile removes the file from the weed cluster.
	RemoveFile(ctx context.Context, dir string) error
}

type deployer struct {
	// config is the config of the weed cluster and etcd cluster.
	config *Config
	// etcd is the etcd cluster.
	etcd Etcd
	// client is the etcd client.
	client Client
	// weedMaster is the weed master node.
	weedMaster Master
	// weedVolume is the weed volume node.
	weedVolume Volume
}

func (d *deployer) GetWeedMasterList(ctx context.Context) ([]string, error) {
	return d.client.GetService("weed-master")
}

func (d *deployer) CreateEtcdCluster(ctx context.Context) error {
	// prepare etcd
	err := d.etcdPrepare()
	if err != nil {
		return err
	}
	// start etcd
	err = d.etcd.Start(ctx, d.config.BinDir+"/etcd")
	if err != nil {
		return err
	}
	// check etcd health
	if ok := d.etcd.IsRunning(ctx); !ok {
		return fmt.Errorf("etcd is not running")
	}
	// new client
	etcdClient, err := NewClient(d.config.MasterIP)
	if err != nil {
		return err
	}
	d.client = etcdClient
	return nil
}

func (d *deployer) downloadEtcd() error {
	url, err := etcdDownloadURL()
	if err != nil {
		return err
	}
	//download
	err = downloadFile(url, EtcdDestination)
	if err != nil {
		return err
	}
	etcdDirName := fmt.Sprintf("%s-%s-%s-%s", EtcdArtifactType, EtcdVersion, runtime.GOOS, runtime.GOARCH)
	err = exec.Command("tar", "-xvf", EtcdDestination, "-C", extractFolder).Run()
	if err != nil {
		return err
	}
	err = os.Rename(path.Join(extractFolder, etcdDirName+"/etcd"), path.Join(d.config.BinDir, EtcdBinName))
	if err != nil {
		return err
	}
	return os.Rename(path.Join(extractFolder, etcdDirName+"/etcdctl"), path.Join(d.config.BinDir, EtcdctlBinName))
}

func (d *deployer) downloadWeed() error {
	url, err := weedDownloadURL()
	if err != nil {
		return err
	}
	err = downloadFile(url, WeedDestination)
	if err != nil {
		return err
	}
	return exec.Command("tar", "-xvf", WeedDestination, "-C", d.config.BinDir).Run()
}

func (d *deployer) etcdPrepare() error {
	var (
		etcdDirs = []string{d.config.DataDir, d.config.LogDir, d.config.PidDir, d.config.BinDir}
	)
	for _, dir := range etcdDirs {
		if err := CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}
	// download etcd
	return d.downloadEtcd()
	// TODO scp etcd binary file to other nodes
}

func (d *deployer) weedMasterPrepare() error {
	var weedMasterDirs []string
	if len(d.config.MasterIP) < 3 {
		d.config.NeedMoreLocalNode = true
		weedMasterPortList := make([]int, 0)
		weedMDirList := make([]string, 0)
		weedMasterList := make([]string, 0)
		port := d.config.WeedMasterPort
		for i := 0; i < 3; i++ {
			for {
				ok := checkPort(port)
				if ok {
					weedMasterPortList = append(weedMasterPortList, port)
					weedMDirList = append(weedMDirList, d.config.WeedMasterDir+fmt.Sprintf("/%d", port))
					weedMasterList = append(weedMasterList, d.config.CurrentIP+fmt.Sprintf(":%d", port))
					port++
					break
				} else {
					port++
				}
			}
		}
		weedMasterDirs = weedMDirList
		d.config.weedMasterPortList = weedMasterPortList
		d.config.weedMDirList = weedMDirList
		d.config.weedMasterList = weedMasterList
	} else {
		weedMasterDirs = []string{d.config.WeedMasterDir}
	}
	for _, dir := range weedMasterDirs {
		if err := CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}
	// download weed binary file
	if checkBinFile(d.config.BinDir + "/weed") {
		return nil
	}
	return d.downloadWeed()
}

func (d *deployer) weedVolumePrepare() error {
	var weedVolumeDirs []string
	if len(d.config.VolumeIP) < 3 {
		d.config.NeedMoreLocalNode = true
		weedVolumePortList := make([]int, 0)
		weedVDirList := make([]string, 0)
		port := d.config.WeedVolumePort
		for i := 0; i < 3; i++ {
			for {
				ok := checkPort(port)
				if ok {
					weedVolumePortList = append(weedVolumePortList, port)
					weedVDirList = append(weedVDirList, d.config.WeedVolumeDir+fmt.Sprintf("/%d", port))
					port++
					break
				} else {
					port++
				}
			}
		}
		weedVolumeDirs = weedVDirList
		d.config.weedVolumePortList = weedVolumePortList
		d.config.weedVDirList = weedVDirList
	} else {
		weedVolumeDirs = []string{d.config.WeedVolumeDir}
	}
	for _, dir := range weedVolumeDirs {
		if err := CreateDirIfNotExists(dir); err != nil {
			return err
		}
	}
	if d.config.NeedMoreLocalNode {
		weedVolume := NewWeedVolume(d.config, d.config.weedMasterList)
		d.weedVolume = weedVolume
	} else {
		weedVolume := NewWeedVolume(d.config, d.config.MasterIP)
		d.weedVolume = weedVolume
	}
	return nil
}

func (d *deployer) DeleteEtcdCluster(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (d *deployer) CreateWeedCluster(ctx context.Context) error {
	// prepare weed master
	err := d.weedMasterPrepare()
	if err != nil {
		return err
	}
	// start weed master
	err = d.weedMaster.Start(ctx, d.config.BinDir+"/weed")
	if err != nil {
		return err
	}
	// check weed master health
	ok := d.weedMaster.IsRunning(ctx)
	if !ok {
		return errors.New("weed master is not running")
	}
	// prepare weed volume
	err = d.weedVolumePrepare()
	if err != nil {
		return err
	}
	err = d.weedVolume.Start(ctx, d.config.BinDir+"/weed")
	if err != nil {
		return err
	}
	// check weed volume health
	ok = d.weedVolume.IsRunning(ctx)
	if !ok {
		return errors.New("weed volume is not running")
	}
	// register service to etcd cluster
	if d.config.NeedMoreLocalNode {
		for _, weedMaster := range d.config.weedMasterList {
			err = d.client.RegisterService("weed-master", weedMaster)
			if err != nil {
				d.cleanService(d.config.weedMasterList)
				return err
			}
		}
	} else {
		for _, masterIp := range d.config.MasterIP {
			err = d.client.RegisterService("weed-master", masterIp)
			if err != nil {
				d.cleanService(d.config.MasterIP)
				return err
			}
		}
	}
	return nil
}

func (d *deployer) cleanService(list []string) {
	for _, l := range list {
		_ = d.client.UnRegisterService("weed-master", l)
	}
}

func (d *deployer) DeleteWeedCluster(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (d *deployer) UploadFile(ctx context.Context, dir string) error {
	masterList, err := d.GetWeedMasterList(ctx)
	if err != nil {
		return err
	}
	for _, m := range masterList {
		resp, err := d.weedMaster.UploadFile(ctx, m, dir)
		if err != nil {
			continue
		}
		// upload resp to etcd
		bytes, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		err = d.client.Put(dir, string(bytes))
		if err != nil {
			continue
		}
		return nil
	}
	return errors.New("cannot upload file to weed cluster")
}

func (d *deployer) DownloadFile(ctx context.Context, dir string, outputDir string) error {
	masterList, err := d.GetWeedMasterList(ctx)
	if err != nil {
		return err
	}
	// get fid
	fid, err := d.client.Get(dir)
	if err != nil {
		return err
	}
	var resp UploadFileResponse
	err = json.Unmarshal([]byte(fid), &resp)
	if err != nil {
		return err
	}
	for _, m := range masterList {
		err = d.weedMaster.DownloadFile(ctx, m, resp.Fid, outputDir)
		if err != nil {
			continue
		}
		return nil
	}
	return errors.New("cannot download file from weed cluster")
}

func (d *deployer) RemoveFile(ctx context.Context, dir string) error {
	//TODO implement me
	panic("implement me")
}

func NewDeployer(config *Config) Deployer {
	check(config)
	return &deployer{
		config:     config,
		etcd:       NewEtcd(config),
		weedMaster: NewMaster(config),
	}
}

func check(config *Config) {
	// check config add set default value if not set
	if config.LogDir == "" {
		config.LogDir = "/tmp/log"
	}
	if config.DataDir == "" {
		config.DataDir = "/tmp/data"
	}
	if config.PidDir == "" {
		config.PidDir = "/tmp/pid"
	}
	if config.BinDir == "" {
		config.BinDir = "/tmp/bin"
	}
	if config.EtcdConfigPath == "" {
		config.EtcdConfigPath = "/tmp/etcd.conf"
	}
	if config.CurrentIP == "" {
		config.CurrentIP = "127.0.0.1"
	}
	if config.PeerPort == 0 {
		config.PeerPort = 2380
	}
	if config.ClientPort == 0 {
		config.ClientPort = 2379
	}
	if config.WeedMasterPort == 0 {
		config.WeedMasterPort = 9333
	}
	if config.WeedVolumePort == 0 {
		config.WeedVolumePort = 8080
	}
	if config.WeedMasterDir == "" {
		config.WeedMasterDir = "/tmp/weed-master"
	}
	if config.WeedVolumeDir == "" {
		config.WeedVolumeDir = "/tmp/weed-volume"
	}
	if config.DefaultReplication == "" {
		config.DefaultReplication = "003"
	}
	if config.WeedLogDir == "" {
		config.WeedLogDir = "/tmp/weed-log"
	}
	if len(config.MasterIP) == 0 {
		logrus.Error("master ip list is empty")
		os.Exit(1)
	}
	if len(config.VolumeIP) == 0 {
		logrus.Error("volume ip list is empty")
		os.Exit(1)
	}
	//check if exist tar file
	_, err := os.Stat(WeedDestination)
	if err == nil {
		_ = os.RemoveAll(WeedDestination)
	}
	_, err = os.Stat(EtcdDestination)
	if err == nil {
		_ = os.RemoveAll(EtcdDestination)
	}

	// test
	_, err = os.Stat(config.BinDir)
	if err == nil {
		_ = os.RemoveAll(config.BinDir)
	}
	_, err = os.Stat(config.DataDir)
	if err == nil {
		_ = os.RemoveAll(config.DataDir)
	}
	_, err = os.Stat(config.LogDir)
	if err == nil {
		_ = os.RemoveAll(config.LogDir)
	}
	_, err = os.Stat(config.PidDir)
	if err == nil {
		_ = os.RemoveAll(config.PidDir)
	}
}
