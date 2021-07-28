package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils/ssh"

	"github.com/alibaba/sealer/rpccall"

	"google.golang.org/grpc/backoff"

	"github.com/alibaba/sealer/rpccall/api"
	"github.com/alibaba/sealer/rpccall/baseapi/oscall"
	"github.com/alibaba/sealer/rpccall/impl"

	"google.golang.org/grpc"
)

type client struct {
	connMux   sync.Mutex
	conn      *grpc.ClientConn
	connector func() (*grpc.ClientConn, error)
}

type handler struct {
	clients map[string]api.Client
	mu      sync.Mutex
}

var staticHandler *handler

func (c *client) OSCallSvc() api.OSCall {
	c.connMux.Lock()
	defer c.connMux.Unlock()
	return impl.NewRemoteOSCall(oscall.NewOSCallClient(c.conn))
}

func (c *client) HealthCheckSvc() grpc_health_v1.HealthClient {
	c.connMux.Lock()
	defer c.connMux.Unlock()
	return grpc_health_v1.NewHealthClient(c.conn)
}

func (c *client) IsServing(ctx context.Context) (bool, error) {
	c.connMux.Lock()
	defer c.connMux.Unlock()
	res, err := c.HealthCheckSvc().Check(ctx, &grpc_health_v1.HealthCheckRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return false, err
	}
	return res.Status == grpc_health_v1.HealthCheckResponse_SERVING, nil
}

func RemoteHandler(remoteHost string, ssh ssh.Interface, opts ...Option) (api.Client, error) {
	staticHandler.mu.Lock()
	defer staticHandler.mu.Unlock()

	var err error
	hst := host(remoteHost)
	c, ok := staticHandler.clients[hst.ip()]
	if ok {
		return c, nil
	}

	isExist := ssh.IsFileExist(remoteHost, common.RemoteSealerPath)
	if !isExist {
		logger.Info("sending sealer to %s", remoteHost)
		err = ssh.Copy(remoteHost, common.RemoteSealerPath, common.RemoteSealerPath)
		if err != nil {
			return nil, fmt.Errorf("failed to send sealer to %s, err: %v", remoteHost, err)
		}

		//err = ssh.CmdAsync(remoteHost, "sealer daemon")
		//logger.Warn("failed to execute sealer daemon, err: %s", err)
	} else {
		//err = ssh.CmdAsync(remoteHost, "sealer daemon")

	}

	go func() {
		err = ssh.CmdAsync(remoteHost, "sealer daemon")
		logger.Warn("failed to execute sealer daemon, err: %s", err)
	}()

	c, err = newClient(fmt.Sprintf("%s:%d", hst.ip(), rpccall.DefaultPort), opts...)
	if err != nil {
		return nil, err
	}

	staticHandler.clients[hst.ip()] = c
	return c, nil
}

func newClient(address string, opts ...Option) (api.Client, error) {
	if address == "" {
		return nil, fmt.Errorf("failed to new grpc clent, address shouldn't be empty")
	}

	opt := clientOpt{}
	for _, o := range opts {
		err := o(&opt)
		if err != nil {
			return nil, err
		}
	}

	if opt.timeout == 0 {
		opt.timeout = 10 * time.Second
	}

	c := &client{}
	defaultConfig := backoff.DefaultConfig
	defaultConfig.MaxDelay = 3 * time.Second
	connectParams := grpc.ConnectParams{
		Backoff: defaultConfig,
	}
	dopts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithConnectParams(connectParams),
	}

	connector := func() (*grpc.ClientConn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), opt.timeout)
		defer cancel()
		conn, err := grpc.DialContext(ctx, address, dopts...)
		if err != nil {
			return nil, fmt.Errorf("failed to dial address %s, err: %v", address, err)
		}
		return conn, nil
	}

	conn, err := connector()
	if err != nil {
		return nil, err
	}
	c.conn, c.connector = conn, connector
	return c, nil
}

func init() {
	staticHandler = &handler{
		clients: map[string]api.Client{},
	}
}
