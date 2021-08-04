package plugin

import (
	ctx "context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/snapshot"
	"go.uber.org/zap"

	"github.com/alibaba/sealer/logger"
)

type EtcdBackupPlugin struct {
	name    string
	backDir string
}

func (e EtcdBackupPlugin) Run(context Context, phase Phase) {
	//Temporary use of local certificate files for testing
	var (
		dialTimeout = 5 * time.Second
		endpoints   = []string{"https://172.17.189.108:2379"}
		etcdCert    = "/Users/liutao/fsdownload/etcd/healthcheck-client.crt"
		etcdCertKey = "/Users/liutao/fsdownload/etcd/healthcheck-client.key"
		etcdCa      = "/Users/liutao/fsdownload/etcd/ca.crt"
	)

	// 创建连接-TLS
	cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
	if err != nil {
		logger.Error("cacert or key file is not exist, err:%v\n", err)
		os.Exit(1)
	}

	caData, err := ioutil.ReadFile(etcdCa)
	if err != nil {
		logger.Error("ca certificate reading failed, err:%v\n", err)
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
		logger.Error("connect to etcd failed, err:%v\n", err)
		os.Exit(1)
	}

	fmt.Println("connect to etcd success")

	defer cli.Close()

	lg, err := zap.NewProduction()
	if err != nil {
		logger.Error("get lg error, err:%v\n", err)
		os.Exit(1)
	}

	ctx, cancel := ctx.WithCancel(ctx.Background())
	defer cancel()

	var dbPath = fmt.Sprintf("%s/%s", e.backDir, e.name)
	if err := snapshot.Save(ctx, lg, cfg, dbPath); err != nil {
		logger.Error("snapshot save err: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Snapshot saved at %s\n", dbPath)
}
