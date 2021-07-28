package api

import "context"

type Client interface {
	OSCallSvc() OSCall

	IsServing(ctx context.Context) (bool, error)
}
