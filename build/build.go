package build

import "github.com/alibaba/sealer/common"

type Interface interface {
	Build(name string, context string, kubefileName string) error
}

func NewBuilder(config *Config, builderType string) Interface {
	if builderType == common.LocalBuild {
		return NewLocalBuilder(config)
	}
	return NewCloudBuilder(config)
}
