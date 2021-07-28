package healthcheck

import (
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type server struct {
	*health.Server
}

func NewServer() grpc_health_v1.HealthServer {
	return server{
		Server: health.NewServer(),
	}
}
