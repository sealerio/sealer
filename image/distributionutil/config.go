package distributionutil

import (
	"github.com/alibaba/sealer/image/store"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/progress"
)

type Config struct {
	LayerStore     store.LayerStore
	ProgressOutput progress.Output
	AuthInfo       types.AuthConfig
}
