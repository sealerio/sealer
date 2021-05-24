package build

import "github.com/alibaba/sealer/common"

type Interface interface {
	Build(name string, context string, kubefileName string) error
}

func NewBuilder(config *Config) (Interface, error) {
	if config.BuildType == common.LocalBuild {
		return NewLocalBuilder(config)
	}
	return NewCloudBuilder(config)
}
