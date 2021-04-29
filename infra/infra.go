package infra

import (
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
)

type Interface interface {
	// Apply apply iaas resources and save metadata info like vpc instance id to cluster status
	// https://github.com/fanux/sealgate/tree/master/cloud
	Apply() error
}

func NewDefaultProvider(cluster *v1.Cluster) Interface {
	switch cluster.Spec.Provider {
	case AliCloud:
		config := new(Config)
		err := GetAKSKFromEnv(config)
		if err != nil {
			logger.Error(err)
			return nil
		}
		aliProvider := new(AliProvider)
		aliProvider.Config = *config
		aliProvider.Cluster = cluster
		err = aliProvider.NewClient()
		if err != nil {
			logger.Error(err)
		}
		return aliProvider
	default:
		return nil
	}

}
