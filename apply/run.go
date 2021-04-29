package apply

import (
	"fmt"
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"gitlab.alibaba-inc.com/seadent/pkg/image"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	"net"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
)

type ClusterArgs struct {
	cluster    *v1.Cluster
	nodeArgs   string
	masterArgs string
}

func IsNumber(args string) bool {
	_, err := strconv.Atoi(args)
	return err == nil
}

func IsIpList(args string) bool {
	ipList := strings.Split(args, ",")

	for _, i := range ipList {
		ip := net.ParseIP(i)
		if ip == nil {
			return false
		}
	}
	return true
}

func (c *ClusterArgs) SetClusterArgs() error {
	if IsNumber(c.masterArgs) && IsNumber(c.nodeArgs) {
		c.cluster.Spec.Masters.Count = c.masterArgs
		c.cluster.Spec.Nodes.Count = c.nodeArgs
		c.cluster.Spec.Provider = common.ALI_CLOUD
		return nil
	}
	if IsIpList(c.masterArgs) && IsIpList(c.nodeArgs) {
		c.cluster.Spec.Masters.IPList = strings.Split(c.masterArgs, ",")
		c.cluster.Spec.Nodes.IPList = strings.Split(c.nodeArgs, ",")
		c.cluster.Spec.Provider = common.BAREMETAL
		return nil
	}
	return fmt.Errorf("enter true iplist or count")
}

func GetClusterFileByImageName(imageName string) (cluster *v1.Cluster, err error) {
	clusterFile := image.GetClusterFileFromImageManifest(imageName)
	if clusterFile == "" {
		return nil, fmt.Errorf("failed to found Clusterfile")
	}
	if err := yaml.Unmarshal([]byte(clusterFile), &cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func NewApplierFromArgs(imageName string, masterArgs, nodeArgs string) (Interface, error) {
	cluster, err := GetClusterFileByImageName(imageName)
	if err != nil {
		return nil, err
	}
	if nodeArgs == "" && masterArgs == "" {
		return NewApplier(cluster), nil
	}
	c := &ClusterArgs{
		cluster:    cluster,
		nodeArgs:   nodeArgs,
		masterArgs: masterArgs,
	}
	if err := c.SetClusterArgs(); err != nil {
		return nil, err
	}
	return NewApplier(c.cluster), nil
}
