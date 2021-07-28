package server

import (
	"fmt"
	"log"
	"net"

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/rpccall/server/healthcheck"

	"github.com/alibaba/sealer/rpccall"

	oscallpb "github.com/alibaba/sealer/rpccall/baseapi/oscall"
	"github.com/alibaba/sealer/rpccall/server/oscall"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	serverOption
}

func NewServer(opts ...Option) (*Server, error) {
	var opt serverOption
	for _, o := range opts {
		err := o(&opt)
		if err != nil {
			return nil, err
		}
	}

	if opt.port == 0 {
		opt.port = rpccall.DefaultPort
	}

	server := Server{
		opt,
	}
	return &server, nil
}

func (s *Server) Serve() error {
	// fix port currently
	var stopCh = make(chan struct{})
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", rpccall.DefaultPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	oscallpb.RegisterOSCallServer(grpcServer, oscall.NewServer())
	grpc_health_v1.RegisterHealthServer(grpcServer, healthcheck.NewServer())
	go func() {
		err = grpcServer.Serve(lis)
		logger.Fatal(err)
		stopCh <- struct{}{}
	}()

	logger.Info("sealer rpc server listening on %d", rpccall.DefaultPort)
	<-stopCh
	return nil
}
