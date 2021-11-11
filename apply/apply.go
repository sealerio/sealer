package apply

import (
	"fmt"

	"github.com/alibaba/sealer/apply/mode"

	"github.com/alibaba/sealer/client/k8s"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

func NewApplierFromFile(clusterfile string) (mode.Interface, error) {
	clusters, err := utils.DecodeCluster(clusterfile)
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return nil, fmt.Errorf("failed to found cluster from %s", clusterfile)
	}
	if len(clusters) > 1 {
		return nil, fmt.Errorf("multiple clusters exist in the Clusterfile")
	}
	cluster := &clusters[0]
	cluster.SetAnnotations(common.ClusterfileName, clusterfile)
	return NewApplier(cluster)
}

func NewApplier(cluster *v1.Cluster) (mode.Interface, error) {
	switch cluster.Spec.Provider {
	case common.AliCloud:
		return NewAliCloudProvider(cluster)
	case common.CONTAINER:
		return NewAliCloudProvider(cluster)
	}
	return NewDefaultApplier(cluster)
}

func NewAliCloudProvider(cluster *v1.Cluster) (mode.Interface, error) {
	return &mode.CloudApplier{
		ClusterDesired: cluster,
	}, nil
}

func NewDefaultApplier(cluster *v1.Cluster) (mode.Interface, error) {
	imgSvc, err := image.NewImageService()
	if err != nil {
		return nil, err
	}

	fs, err := filesystem.NewFilesystem()
	if err != nil {
		return nil, err
	}

	k8sClient, err := k8s.Newk8sClient()
	if err != nil {
		logger.Warn(err)
	}

	return &mode.Applier{
		ClusterDesired: cluster,
		ImageManager:   imgSvc,
		FileSystem:     fs,
		Client:         k8sClient,
	}, nil
}
