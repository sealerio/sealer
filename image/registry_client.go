package image

import (
	"context"

	"github.com/alibaba/sealer/logger"
	pkgutils "github.com/alibaba/sealer/utils"
	"github.com/pkg/errors"

	"github.com/alibaba/sealer/registry"
	"github.com/docker/docker/api/types"
)

func initRegistry(hostname string) (*registry.Registry, error) {
	var (
		authInfo types.AuthConfig
		err      error
		reg      *registry.Registry
	)

	authInfo, err = pkgutils.GetDockerAuthInfoFromDocker(hostname)
	if err != nil {
		logger.Warn("failed to get auth info for %s, err: %s", hostname, err)
	}

	reg, err = fetchRegistryClient(authInfo)
	if err != nil {
		err = errors.Wrap(err, "failed to fetch registry client")
		return nil, err
	}
	return reg, err
}

//fetch https and http registry client
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
