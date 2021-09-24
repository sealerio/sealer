// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package container

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"os"
	"strconv"
	"time"

	"github.com/alibaba/sealer/utils/ssh"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/docker/docker/api/types/mount"
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
	ID          string
	Type        string
	DefaultName string
}

type Container struct {
	ContainerID       string
	ContainerName     string
	ContainerHostName string
	ContainerIP       string
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
	ToDeleteIPList []string
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
	if c.Cluster.Annotations[NETWROKID] == "" || c.Cluster.Annotations[IMAGEID] == "" {
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
	if c.Cluster.Annotations[NETWROKID] == "" || c.Cluster.Annotations[IMAGEID] == "" {
		return fmt.Errorf("network %s or image %s not found", c.NetworkResource.DefaultName, c.ImageResource.DefaultName)
	}
	currentMasterNum := len(c.Cluster.Spec.Masters.IPList)

	num, list, _ := getDiff(c.Cluster.Spec.Masters)
	masterApplyResult := &ApplyResult{
		ToJoinNumber:   num,
		ToDeleteIPList: list,
		Role:           MASTER,
	}
	num, list, _ = getDiff(c.Cluster.Spec.Nodes)
	nodeApplyResult := &ApplyResult{
		ToJoinNumber:   num,
		ToDeleteIPList: list,
		Role:           NODE,
	}
	//Abnormal scene :master number must > 0
	if currentMasterNum+masterApplyResult.ToJoinNumber-len(masterApplyResult.ToDeleteIPList) <= 0 {
		return fmt.Errorf("master number can not be 0")
	}
	logger.Info("master apply result: ToJoinNumber %d, ToDeleteIpList : %s",
		masterApplyResult.ToJoinNumber, masterApplyResult.ToDeleteIPList)

	logger.Info("node apply result: ToJoinNumber %d, ToDeleteIpList : %s",
		nodeApplyResult.ToJoinNumber, nodeApplyResult.ToDeleteIPList)

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
			joinIPList, err := c.applyToJoin(result.ToJoinNumber, result.Role)
			if err != nil {
				return err
			}
			c.Cluster.Spec.Masters.IPList = append(c.Cluster.Spec.Masters.IPList, joinIPList...)
		}
		if len(result.ToDeleteIPList) > 0 {
			err := c.applyToDelete(result.ToDeleteIPList)
			if err != nil {
				return err
			}
			c.Cluster.Spec.Masters.IPList = c.Cluster.Spec.Masters.IPList[:len(c.Cluster.Spec.Masters.IPList)-
				len(result.ToDeleteIPList)]
		}
	}

	if result.Role == NODE {
		if result.ToJoinNumber > 0 {
			joinIPList, err := c.applyToJoin(result.ToJoinNumber, result.Role)
			if err != nil {
				return err
			}
			c.Cluster.Spec.Nodes.IPList = append(c.Cluster.Spec.Nodes.IPList, joinIPList...)
		}
		if len(result.ToDeleteIPList) > 0 {
			err := c.applyToDelete(result.ToDeleteIPList)
			if err != nil {
				return err
			}
			c.Cluster.Spec.Nodes.IPList = c.Cluster.Spec.Nodes.IPList[:len(c.Cluster.Spec.Nodes.IPList)-
				len(result.ToDeleteIPList)]
		}
	}
	return nil
}

func (c *DockerProvider) applyToJoin(toJoinNumber int, role string) ([]string, error) {
	// run container and return append ip list
	var toJoinIPList []string
	for i := 0; i < toJoinNumber; i++ {
		name := fmt.Sprintf("sealer-%s-%s", role, utils.GenUniqueID(10))
		opts := &CreateOptsForContainer{
			ContainerHostName: name,
			ContainerName:     name,
			ContainerLabel: map[string]string{
				RoleLabel: role,
			},
		}
		if len(c.Cluster.Spec.Masters.IPList) == 0 && i == 0 {
			opts.ContainerLabel[RoleLabelMaster] = "true"
			opts.IsMaster0 = true
		}

		containerID, err := c.RunContainer(opts)
		if err != nil {
			return toJoinIPList, fmt.Errorf("failed to create container %s,error is %v", opts.ContainerName, err)
		}
		time.Sleep(3 * time.Second)
		info, err := c.GetContainerInfo(containerID)
		if err != nil {
			return toJoinIPList, fmt.Errorf("failed to get container info of %s,error is %v", containerID, err)
		}

		err = c.changeDefaultPasswd(info.ContainerIP)
		if err != nil {
			return nil, fmt.Errorf("failed to change container password of %s,error is %v", containerID, err)
		}
		toJoinIPList = append(toJoinIPList, info.ContainerIP)
	}

	return toJoinIPList, nil
}

func (c *DockerProvider) changeDefaultPasswd(containerIP string) error {
	if c.Cluster.Spec.SSH.Passwd == "" {
		return nil
	}

	if c.Cluster.Spec.SSH.Passwd == DefaultPassword {
		return nil
	}

	user := "root"
	if c.Cluster.Spec.SSH.User != "" {
		user = c.Cluster.Spec.SSH.User
	}
	sshClient := &ssh.SSH{
		User:     user,
		Password: DefaultPassword,
	}

	cmd := fmt.Sprintf(ChangePasswordCmd, c.Cluster.Spec.SSH.Passwd)
	return c.RunSSHCMDInContainer(sshClient, containerIP, cmd)
}
func (c *DockerProvider) applyToDelete(deleteIPList []string) error {
	// delete container and return deleted ip list
	for _, ip := range deleteIPList {
		id, err := c.GetContainerIDByIP(ip)
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
		return fmt.Errorf("failed to get docker server, please check docker server running status")
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

	if !utils.IsFileExist(DockerHost) && os.Getenv("DOCKER_HOST") == "" {
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
	c.Cluster.Annotations[NETWROKID] = c.NetworkResource.ID
	c.Cluster.Annotations[IMAGEID] = c.ImageResource.ID
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
		id, err := c.GetContainerIDByIP(ip)
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

	deleteNetErr := c.DeleteNetworkResource(c.Cluster.Annotations[NETWROKID])

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
	if err != nil {
		return nil, err
	}
	return &DockerProvider{
		Ctx:          ctx,
		DockerClient: cli,
		Cluster:      cluster,
		NetworkResource: &Resource{
			Type:        ResourceNetwork,
			DefaultName: DefaultNetworkName,
		},
		ImageResource: &Resource{
			Type:        ResourceImage,
			DefaultName: DefaultImageName,
		},
	}, nil
}
