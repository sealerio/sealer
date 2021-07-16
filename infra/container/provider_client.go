package container

import "C"
import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type DockerProvider struct {
	DockerClient    *client.Client
	Ctx             context.Context
	Cluster         *v1.Cluster
	ImageResource   *Resource
	NetworkResource *Resource
	Containers      []Container
}

type Resource struct {
	Id          string
	Type        string
	DefaultName string
}

type Container struct {
	ContainerID       string
	ContainerName     string
	ContainerHostName string
	ContainerIp       string
	Status            string
	ContainerLabel    map[string]string
}

type CreateOptsForContainer struct {
	ContainerName     string
	ContainerHostName string
	ContainerLabel    map[string]string
	Mount             *mount.Mount
	IsMaster0         bool
}

type DockerInfo struct {
	CgroupDriver    string
	CgroupVersion   string
	StorageDriver   string
	MemoryLimit     bool
	PidsLimit       bool
	CPUShares       bool
	CPUNumber       int
	SecurityOptions []string
}

type ApplyResult struct {
	ToJoinNumber   int
	ToDeleteIpList []string
	Role           string
}

func (c *DockerProvider) Apply() error {
	/*
		1,run sys check
		2,prepare base resource such as:network and image
		3,create container
		4,write infra info to cluster
		5,clean up : delete container,umount cluster ,remove mounted cluster.delete time stamp
	*/
	if c.Cluster.Annotations == nil {
		c.Cluster.Annotations = make(map[string]string)
	}
	// delete apply
	if c.Cluster.DeletionTimestamp != nil {
		logger.Info("deletion timestamp not nil, will clear infra")
		return c.CleanUp()
	}
	// new apply
	if c.Cluster.Annotations[NETWROK_ID] == "" || c.Cluster.Annotations[IMAGE_ID] == "" {
		var pipLine []func() error
		pipLine = append(pipLine,
			c.CheckServerInfo,
			c.PrepareBaseResource,
			c.ReconcileContainer)

		for _, f := range pipLine {
			if err := f(); err != nil {
				return err
			}
		}
	}
	// change apply: scale up or scale down,count!=len(iplist)
	if c.Cluster.Spec.Masters.Count != strconv.Itoa(len(c.Cluster.Spec.Masters.IPList)) ||
		c.Cluster.Spec.Nodes.Count != strconv.Itoa(len(c.Cluster.Spec.Nodes.IPList)) {
		return c.ReconcileContainer()
	}

	return nil
}

func (c *DockerProvider) ReconcileContainer() error {
	// check image id and network id!= nil . if so return error
	// scale up: apply diff container, append ip list.
	// scale down: delete diff container by id,delete ip list. if no container,need do cleanup
	if c.Cluster.Annotations[NETWROK_ID] == "" || c.Cluster.Annotations[IMAGE_ID] == "" {
		return fmt.Errorf("network %s or image %s not found", c.NetworkResource.DefaultName, c.ImageResource.DefaultName)
	}
	currentMasterNum := len(c.Cluster.Spec.Masters.IPList)

	num, list, _ := getDiff(c.Cluster.Spec.Masters)
	masterApplyResult := &ApplyResult{
		ToJoinNumber:   num,
		ToDeleteIpList: list,
		Role:           MASTER,
	}
	num, list, _ = getDiff(c.Cluster.Spec.Nodes)
	nodeApplyResult := &ApplyResult{
		ToJoinNumber:   num,
		ToDeleteIpList: list,
		Role:           NODE,
	}
	//Abnormal scene :master number must > 0
	if currentMasterNum+masterApplyResult.ToJoinNumber-len(masterApplyResult.ToDeleteIpList) <= 0 {
		return fmt.Errorf("master number can not be 0")
	}
	logger.Info("master apply result: ToJoinNumber %d, ToDeleteIpList : %s",
		masterApplyResult.ToJoinNumber, masterApplyResult.ToDeleteIpList)

	logger.Info("node apply result: ToJoinNumber %d, ToDeleteIpList : %s",
		nodeApplyResult.ToJoinNumber, nodeApplyResult.ToDeleteIpList)

	if err := c.applyResult(masterApplyResult); err != nil {
		return err
	}
	if err := c.applyResult(nodeApplyResult); err != nil {
		return err
	}
	return nil

}

func (c *DockerProvider) applyResult(result *ApplyResult) error {
	// create or delete an update iplist
	if result.Role == MASTER {
		if result.ToJoinNumber > 0 {
			joinIpList, err := c.applyToJoin(result.ToJoinNumber, result.Role)
			if err != nil {
				return err
			}
			c.Cluster.Spec.Masters.IPList = append(c.Cluster.Spec.Masters.IPList, joinIpList...)
		}
		if len(result.ToDeleteIpList) > 0 {
			err := c.applyToDelete(result.ToDeleteIpList)
			if err != nil {
				return err
			}
			c.Cluster.Spec.Masters.IPList = c.Cluster.Spec.Masters.IPList[:len(c.Cluster.Spec.Masters.IPList)-
				len(result.ToDeleteIpList)]
		}

	}

	if result.Role == NODE {
		if result.ToJoinNumber > 0 {
			joinIpList, err := c.applyToJoin(result.ToJoinNumber, result.Role)
			if err != nil {
				return err
			}
			c.Cluster.Spec.Nodes.IPList = append(c.Cluster.Spec.Nodes.IPList, joinIpList...)
		}
		if len(result.ToDeleteIpList) > 0 {
			err := c.applyToDelete(result.ToDeleteIpList)
			if err != nil {
				return err
			}
			c.Cluster.Spec.Nodes.IPList = c.Cluster.Spec.Nodes.IPList[:len(c.Cluster.Spec.Nodes.IPList)-
				len(result.ToDeleteIpList)]
		}
	}

	return nil
}

func (c *DockerProvider) applyToJoin(toJoinNumber int, role string) ([]string, error) {
	// run container and return append ip list
	var toJoinIpList []string
	for i := 0; i < toJoinNumber; i++ {
		name := fmt.Sprintf("sealer-%s-%s", role, utils.GenUniqueID(10))
		opts := &CreateOptsForContainer{
			ContainerHostName: name,
			ContainerName:     name,
			ContainerLabel: map[string]string{
				CONTAINERLABLE: role,
			},
		}
		if len(c.Cluster.Spec.Masters.IPList) == 0 && i == 0 {
			opts.ContainerLabel[CONTAINERLABLEMASTER] = "true"
			opts.IsMaster0 = true
		}

		containerId, err := c.RunContainer(opts)
		if err != nil {
			return toJoinIpList, fmt.Errorf("failed to create container %s,error is %v", opts.ContainerName, err)
		}
		time.Sleep(3 * time.Second)
		info, err := c.GetContainerInfo(containerId)
		if err != nil {
			return toJoinIpList, fmt.Errorf("failed to get container info of %s,error is %v", containerId, err)
		}

		err = c.changeDefaultPasswd(info.ContainerIp)
		if err != nil {
			return nil, fmt.Errorf("failed to change container password of %s,error is %v", containerId, err)
		}
		toJoinIpList = append(toJoinIpList, info.ContainerIp)
	}

	return toJoinIpList, nil
}

func (c *DockerProvider) changeDefaultPasswd(containerIp string) error {
	if c.Cluster.Spec.SSH.Passwd == "" {
		return nil
	}

	if c.Cluster.Spec.SSH.Passwd == DEFAULT_PASSWORD {
		return nil
	}

	cmd := fmt.Sprintf(CHANGE_PASSWORD_CMD, c.Cluster.Spec.SSH.Passwd)

	user := "root"
	if c.Cluster.Spec.SSH.User != "" {
		user = c.Cluster.Spec.SSH.User
	}
	sshClient := &ssh.SSH{
		User:     user,
		Password: DEFAULT_PASSWORD,
	}

	return c.RunSSHCMDInContainer(sshClient, containerIp, cmd)
}
func (c *DockerProvider) applyToDelete(deleteIpList []string) error {
	// delete container and return deleted ip list
	for _, ip := range deleteIpList {
		id, err := c.GetContainerIdByIp(ip)
		if err != nil {
			return fmt.Errorf("failed to get container id %s while delte it ", ip)
		}
		err = c.RmContainer(id)
		if err != nil {
			return fmt.Errorf("failed to delete container:%s", id)
		}
	}

	return nil
}

func (c *DockerProvider) CheckServerInfo() error {
	/*
		1,rootless docker:do not support rootless docker currently.if support, CgroupVersion must = 2
		2,StorageDriver:overlay2
		3,cpu num >1
		4,docker host : /var/run/docker.sock. set env DOCKER_HOST to override
	*/
	info, err := c.GetServerInfo()
	if err != nil {
		return fmt.Errorf("failed to get docker server, please check docker server runing status")
	}
	if info.StorageDriver != "overlay2" {
		return fmt.Errorf("only support storage driver overlay2 ,but current is :%s", info.StorageDriver)
	}

	if info.CPUNumber <= 1 {
		return fmt.Errorf("cpu number of docker server must greater than 1 ,but current is :%d", info.CPUNumber)
	}

	for _, opt := range info.SecurityOptions {
		if opt == "name=rootless" {
			return fmt.Errorf("do not support rootless docker currently")
		}
	}

	if !utils.IsFileExist(DOCKER_HOST) && os.Getenv("DOCKER_HOST") == "" {
		return fmt.Errorf("sealer user default docker host /var/run/docker.sock, please set env DOCKER_HOST='' to override it")
	}

	return nil
}

func (c *DockerProvider) GetServerInfo() (*DockerInfo, error) {
	sysInfo, err := c.DockerClient.Info(c.Ctx)
	if err != nil {
		return nil, err
	}

	return &DockerInfo{
		CgroupDriver:    sysInfo.CgroupDriver,
		CgroupVersion:   sysInfo.CgroupVersion,
		StorageDriver:   sysInfo.Driver,
		MemoryLimit:     sysInfo.MemoryLimit,
		PidsLimit:       sysInfo.PidsLimit,
		CPUShares:       sysInfo.CPUShares,
		CPUNumber:       sysInfo.NCPU,
		SecurityOptions: sysInfo.SecurityOptions,
	}, nil

}

func (c *DockerProvider) PrepareBaseResource() error {
	// prepare network
	err := c.PrepareNetworkResource()
	if err != nil {
		logger.Error("failed to prepare network resource:", err)
		return err
	}
	// prepare image
	err = c.PrepareImageResource()
	if err != nil {
		logger.Error("failed to prepare image resource:", err)
		return err
	}

	if c.Cluster.Annotations == nil {
		c.Cluster.Annotations = make(map[string]string)
	}
	c.Cluster.Annotations[NETWROK_ID] = c.NetworkResource.Id
	c.Cluster.Annotations[IMAGE_ID] = c.ImageResource.Id
	logger.Info("prepare base image %s and network %s successfully ", c.ImageResource.DefaultName, c.NetworkResource.DefaultName)
	return nil
}

func (c *DockerProvider) CleanUp() error {
	/*	a,clean up container,cleanup image,clean up network
		b,rm -rf /var/lib/sealer/data/my-cluster
	*/
	var iplist []string
	iplist = append(iplist, c.Cluster.Spec.Masters.IPList...)
	iplist = append(iplist, c.Cluster.Spec.Nodes.IPList...)

	for _, ip := range iplist {
		id, err := c.GetContainerIdByIp(ip)
		if err != nil {
			return fmt.Errorf("failed to get container id %s while delte it ", ip)
		}
		err = c.RmContainer(id)
		if err != nil {
			// log it
			logger.Info("failed to delete container:%s", id)
		}
		continue
	}
	utils.CleanDir(common.DefaultClusterBaseDir(c.Cluster.Name))

	deleteNetErr := c.DeleteNetworkResource(c.Cluster.Annotations[NETWROK_ID])

	if deleteNetErr != nil {
		logger.Error("failed to clean up resource: %v", deleteNetErr)
		return nil
	}
	logger.Info("delete network  %s successfully", c.NetworkResource.DefaultName)
	return nil
}

func NewClientWithCluster(cluster *v1.Cluster) (*DockerProvider, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerProvider{
		Ctx:          ctx,
		DockerClient: cli,
		Cluster:      cluster,
		NetworkResource: &Resource{
			Type:        RESOURCE_NETWORK,
			DefaultName: DEFAULT_NETWORK_NAME,
		},
		ImageResource: &Resource{
			Type:        RESOURCE_IMAGE,
			DefaultName: DEFAULT_IMAGE_NAME,
		},
	}, nil
}
