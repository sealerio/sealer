package distributionutil

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/justadogistaken/reg/registry"
)

func fetchRegistryClient(auth types.AuthConfig) (*registry.Registry, error) {
	reg, err := registry.New(context.Background(), auth, registry.Opt{Insecure: true})
	if err == nil {
		return reg, nil
	}
	reg, err = registry.New(context.Background(), auth, registry.Opt{Insecure: true, NonSSL: true})
	if err == nil {
		return reg, nil
	}
	return nil, err
}
