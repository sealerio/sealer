package image_adaptor

import (
	"github.com/sealerio/sealer/pkg/image_adaptor/common"
)

type Interface interface {
	Build(sealerBuildFlags *common.BuildFlags, inputArgs []string) error
}
