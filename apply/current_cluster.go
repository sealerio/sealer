package apply

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.alibaba-inc.com/seadent/pkg/client"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
)

const MasterRoleLabel = "node-role.kubernetes.io/master"

func GetCurrentCluster() (*v1.Cluster, error) {
	return getCurrentNodes()
}

func getCurrentNodes() (*v1.Cluster, error) {
	c, err := client.NewClientSet()
	if err != nil {
		logger.Info("current cluster not found, will create a new cluster %v", err)
		return nil, nil
	}
	nodes, err := client.ListNodes(c)
	if err != nil {
		logger.Info("current cluster nodes not found, will create a new cluster")
		return nil, nil
	}

	cluster := &v1.Cluster{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       v1.ClusterSpec{},
		Status:     v1.ClusterStatus{},
	}

	for _, node := range nodes.Items {
		addr := getNodeAddress(&node)
		if addr == "" {
			continue
		}
		if _, ok := node.Labels[MasterRoleLabel]; ok {
			cluster.Spec.Masters.IPList = append(cluster.Spec.Masters.IPList, addr)
			continue
		}
		cluster.Spec.Nodes.IPList = append(cluster.Spec.Nodes.IPList, addr)
	}

	return cluster, nil
}

func getNodeAddress(node *corev1.Node) string {
	if len(node.Status.Addresses) < 1 {
		return ""
	}
	return node.Status.Addresses[0].Address
}
