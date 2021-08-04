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

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"time"

	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/snapshot"
)

var (
	dialTimeout    = 5 * time.Second
	requestTimeout = 4 * time.Second
	endpoints      = []string{"https://172.17.189.108:2379"}
	etcdCert       = "/Users/liutao/fsdownload/etcd/healthcheck-client.crt"
	etcdCertKey    = "/Users/liutao/fsdownload/etcd/healthcheck-client.key"
	etcdCa         = "/Users/liutao/fsdownload/etcd/ca.crt"
)

func main() {
	// 尝试创建连接-TLS，并合并
	cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
	if err != nil {
		fmt.Printf("cacert or key file is not exist, err:%v\n", err)
		os.Exit(1)
	}

	caData, err := ioutil.ReadFile(etcdCa)
	if err != nil {
		fmt.Printf("ca certificate reading failed, err:%v\n", err)
		os.Exit(1)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)

	_tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}

	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
		TLS:         _tlsConfig,
	}

	cli, err := clientv3.New(cfg)
	if err != nil {
		fmt.Printf("connect to etcd failed, err:%v\n", err)
		os.Exit(1)
	}

	fmt.Println("connect to etcd success")

	defer cli.Close()

	lg, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("get lg error, err:%v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := snapshot.Save(ctx, lg, cfg, "/Users/liutao/fsdownload/etcd/123.snap"); err != nil {
		fmt.Printf("snapshot save err: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Snapshot saved at %s\n", "/Users/liutao/fsdownload/etcd/123.snap")
}
