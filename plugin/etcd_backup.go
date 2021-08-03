package plugin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"go.etcd.io/etcd/clientv3"
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
	// 尝试创建连接-TLS
	cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
	if err != nil {
		fmt.Printf("cert failed, err:%v\n", err)
		return
	}
	caData, err := ioutil.ReadFile(etcdCa)
	if err != nil {
		return
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
		// handle error
		fmt.Printf("connect to etcd failed, err:%v\n", err)
		return
	}

	fmt.Println("connect to etcd success")

	defer cli.Close()

	// put
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	_, err = cli.Put(ctx, "标", "hello")
	cancel()
	if err != nil {
		fmt.Printf("put to etcd failed, err:%v\n", err)
		return
	}

	// get
	ctx, cancel = context.WithTimeout(context.Background(), requestTimeout)
	resp, err := cli.Get(ctx, "标")
	cancel()

	if err != nil {
		fmt.Printf("get from etcd failed, err:%v\n", err)
		return
	}

	for _, kv := range resp.Kvs {
		fmt.Printf("%s:%s\n", kv.Key, kv.Value)
	}
}
