package clusterincontainer

import (
	"github.com/alibaba/sealer/clusterincontainer/providers"
	"github.com/alibaba/sealer/clusterincontainer/providers/docker"
)

type ProviderManager struct {
}

func NewProviderManager() *ProviderManager {
	return &ProviderManager{}
}

func (pm *ProviderManager) CreateDockerProvider() (providers.Provider, error) {
	return docker.NewDockerProvider()
}
