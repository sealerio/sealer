package apply

import (
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
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
	case common.AliCloud:
		return NewAliCloudProvider(cluster)
	}
	return NewDefaultApplier(cluster)
}
