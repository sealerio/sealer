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

package plugin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/snapshot"
	"go.uber.org/zap"

	"github.com/sealerio/sealer/utils/ssh"
	"github.com/sirupsen/logrus"
)

type EtcdBackupPlugin struct {
}

func NewEtcdBackupPlugin() Interface {
	return &EtcdBackupPlugin{}
}

func init() {
	Register(EtcdPlugin, NewEtcdBackupPlugin())
}

func (e EtcdBackupPlugin) Run(context Context, phase Phase) error {
	masterIP, err := getMasterIP(context)
	if err != nil {
		return err
	}

	if err := fetchRemoteCert(context, masterIP); err != nil {
		return err
	}

	cfg, err := connEtcd(masterIP)
	if err != nil {
		return err
	}

	return snapshotEtcd(context.Plugin.Spec.On, cfg)
}

func getMasterIP(context Context) (net.IP, error) {
	masterIPList := context.Cluster.GetMasterIPList()
	if len(masterIPList) == 0 {
		return nil, errors.New("cluster master does not exist")
	}
	return masterIPList[0], nil
}

func fetchRemoteCert(context Context, masterIP net.IP) error {
	certs := []string{"healthcheck-client.crt", "healthcheck-client.key", "ca.crt"}
	for _, cert := range certs {
		sshClient, err := ssh.GetHostSSHClient(masterIP, context.Cluster)
		if err != nil {
			return err
		}
		if err := sshClient.Fetch(masterIP, "/tmp/"+cert, "/etc/kubernetes/pki/etcd/"+cert); err != nil {
			return fmt.Errorf("host %s %s file does not exist, err: %v", masterIP, cert, err)
		}
	}
	return nil
}

func connEtcd(masterIP net.IP) (clientv3.Config, error) {
	const (
		dialTimeout = 5 * time.Second
		etcdCert    = "/tmp/healthcheck-client.crt"
		etcdCertKey = "/tmp/healthcheck-client.key"
		etcdCa      = "/tmp/ca.crt"
	)

	cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
	if err != nil {
		return clientv3.Config{}, fmt.Errorf("failed to load cacert or key file: %v", err)
	}

	caData, err := ioutil.ReadFile(etcdCa)
	if err != nil {
		return clientv3.Config{}, fmt.Errorf("failed to read ca certificate: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)
	// #nosec
	_tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}

	endpoints := []string{fmt.Sprintf("https://%s:2379", masterIP)}
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
		TLS:         _tlsConfig,
	}

	cli, err := clientv3.New(cfg)
	if err != nil {
		return clientv3.Config{}, fmt.Errorf("failed to connect etcd: %v", err)
	}

	logrus.Info("connect to etcd success")

	defer cli.Close()

	return cfg, nil
}

func snapshotEtcd(snapshotPath string, cfg clientv3.Config) error {
	lg, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to get zap logger: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := snapshot.Save(ctx, lg, cfg, snapshotPath); err != nil {
		return fmt.Errorf("failed to save snapshot: %v", err)
	}
	logrus.Infof("Snapshot saved at %s", snapshotPath)

	return nil
}
