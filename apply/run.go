package apply

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"sigs.k8s.io/yaml"
)

type ClusterArgs struct {
	cluster    *v1.Cluster
	imageName  string
	nodeArgs   string
	masterArgs string
	user       string
	passwd     string
	pk         string
	pkPasswd   string
}

func IsNumber(args string) bool {
	_, err := strconv.Atoi(args)
	return err == nil
}

func IsIPList(args string) bool {
	ipList := strings.Split(args, ",")

	for _, i := range ipList {
		if !strings.Contains(i, ":") {
			return net.ParseIP(i) != nil
		}
		if _, err := net.ResolveTCPAddr("tcp", i);err != nil {
			return false
		}
	}
	return true
}

func (c *ClusterArgs) SetClusterArgs() error {
	c.cluster.Spec.Image = c.imageName
	c.cluster.Spec.Provider = common.BAREMETAL
	if IsNumber(c.masterArgs) && (IsNumber(c.nodeArgs) || c.nodeArgs == "") {
		c.cluster.Spec.Masters.Count = c.masterArgs
		c.cluster.Spec.Nodes.Count = c.nodeArgs
		c.cluster.Spec.SSH.Passwd = c.passwd
		c.cluster.Spec.Provider = common.DefaultCloudProvider
		return nil
	}
	if IsIPList(c.masterArgs) && (IsIPList(c.nodeArgs) || c.nodeArgs == "") {
		c.cluster.Spec.Masters.IPList = strings.Split(c.masterArgs, ",")
		if c.nodeArgs != "" {
			c.cluster.Spec.Nodes.IPList = strings.Split(c.nodeArgs, ",")
		}
		c.cluster.Spec.SSH.User = c.user
		c.cluster.Spec.SSH.Passwd = c.passwd
		c.cluster.Spec.SSH.Pk = c.pk
		c.cluster.Spec.SSH.PkPasswd = c.pkPasswd
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

func NewApplierFromArgs(imageName string, runArgs *common.RunArgs) (Interface, error) {
	cluster, err := GetClusterFileByImageName(imageName)
	if err != nil {
		return nil, err
	}
	if runArgs.Nodes == "" && runArgs.Masters == "" {
		return NewApplier(cluster), nil
	}
	c := &ClusterArgs{
		cluster:    cluster,
		imageName:  imageName,
		nodeArgs:   runArgs.Nodes,
		masterArgs: runArgs.Masters,
		user:       runArgs.User,
		passwd:     runArgs.Password,
		pk:         runArgs.Pk,
		pkPasswd:   runArgs.PkPassword,
	}
	if err := c.SetClusterArgs(); err != nil {
		return nil, err
	}
	return NewApplier(c.cluster), nil
}
