package build

import "gitlab.alibaba-inc.com/seadent/pkg/common"

type Interface interface {
	Build(name string, context string, kubefileName string) error
}

func NewBuilder(config *Config, builderType string) Interface {
	if builderType == common.LocalBuild {
		return NewLocalBuilder(config)
	}
	return NewCloudBuilder(config)
}
