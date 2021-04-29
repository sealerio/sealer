package apply

import (
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	"gitlab.alibaba-inc.com/seadent/pkg/utils"
)

type Interface interface {
	Apply() error
	Delete() error
}

func NewApplierFromFile(clusterfile string) Interface {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterfile, cluster); err != nil {
		logger.Error("apply cloud cluster failed", err)
		return nil
	}
	return NewApplier(cluster)
}

func NewApplier(cluster *v1.Cluster) Interface {
	switch cluster.Spec.Provider {
	case common.ALI_CLOUD:
		return NewAliCloudProvider(cluster)
	}
	return NewDefaultApplier(cluster)
}
