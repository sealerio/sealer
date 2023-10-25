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
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/sealerio/sealer/pkg/infra/container/client"
	"github.com/sealerio/sealer/pkg/infra/container/client/docker"
	v1 "github.com/sealerio/sealer/types/api/v1"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/ssh"
	"github.com/sirupsen/logrus"
)

const (
	CONTAINER           = "CONTAINER"
	DockerHost          = "/var/run/docker.sock"
	DefaultPassword     = "Seadent123"
	MASTER              = "master"
	NODE                = "node"
	ChangePasswordCmd   = "echo root:%s | chpasswd" // #nosec
	RoleLabel           = "sealer-io-role"
	RoleLabelMaster     = "sealer-io-role-is-master"
	NetworkName         = "sealer-network"
	ImageName           = "sealerio/sealer-base-image:v1"
	SealerImageRootPath = "/var/lib/sealer"
	// for check rootless docker in info.SecurityOptions
	RootlessDocker = "name=rootless"
	// for check overlay2 StorageDriver in info.StorageDriver
	Overlay2 = "overlay2"
)

type ApplyProvider struct {
	Cluster  *v1.Cluster
	Provider client.ProviderService
}

type ApplyResult struct {
	ToJoinNumber   int
	ToDeleteIPList []net.IP
	Role           string
}

func (a *ApplyProvider) Apply() error {
	// delete apply
	if a.Cluster.DeletionTimestamp != nil {
		logrus.Info("deletion timestamp not nil, will clear infra")
		return a.CleanUp()
	}
	// new apply
	if a.Cluster.Annotations == nil {
		err := a.CheckServerInfo()
		if err != nil {
			return err
		}
		a.Cluster.Annotations = make(map[string]string)
	}
	// change apply: scale up or scale down,count!=len(iplist)
	if a.Cluster.Spec.Masters.Count != strconv.Itoa(len(a.Cluster.Spec.Masters.IPList)) ||
		a.Cluster.Spec.Nodes.Count != strconv.Itoa(len(a.Cluster.Spec.Nodes.IPList)) {
		return a.ReconcileContainer()
	}
	return nil
}

func (a *ApplyProvider) CheckServerInfo() error {
	/*
		1,rootless docker:do not support rootless docker currently.if support, CgroupVersion must = 2
		2,StorageDriver:overlay2
		3,cpu num >1
		4,docker host : /var/run/docker.sock. set env DOCKER_HOST to override
	*/
	info, err := a.Provider.GetServerInfo()
	if err != nil {
		return fmt.Errorf("failed to get docker server, please check docker server running status")
	}

	for _, opt := range info.SecurityOptions {
		if opt == RootlessDocker {
			return fmt.Errorf("do not support rootless docker currently")
		}
	}

	if info.StorageDriver != Overlay2 {
		return fmt.Errorf("only support storage driver overlay2 ,but current is :%s", info.StorageDriver)
	}

	if info.CPUNumber <= 1 {
		return fmt.Errorf("cpu number of docker server must greater than 1 ,but current is :%d", info.CPUNumber)
	}

	if !info.MemoryLimit || !info.PidsLimit || !info.CPUShares {
		return fmt.Errorf("requires setting systemd property \"Delegate=yes\"")
	}

	if !osi.IsFileExist(DockerHost) && os.Getenv("DOCKER_HOST") == "" {
		return fmt.Errorf("sealer user default docker host /var/run/docker.sock, please set env DOCKER_HOST='' to override it")
	}

	return nil
}

func (a *ApplyProvider) ReconcileContainer() error {
	// scale up: apply diff container, append ip list.
	// scale down: delete diff container by id,delete ip list. if no container,need do cleanup
	currentMasterNum := len(a.Cluster.Spec.Masters.IPList)
	num, list, _ := getDiff(a.Cluster.Spec.Masters)
	masterApplyResult := &ApplyResult{
		ToJoinNumber:   num,
		ToDeleteIPList: list,
		Role:           MASTER,
	}
	num, list, _ = getDiff(a.Cluster.Spec.Nodes)
	nodeApplyResult := &ApplyResult{
		ToJoinNumber:   num,
		ToDeleteIPList: list,
		Role:           NODE,
	}
	//Abnormal scene :master number must > 0
	if currentMasterNum+masterApplyResult.ToJoinNumber-len(masterApplyResult.ToDeleteIPList) <= 0 {
		return fmt.Errorf("master number can not be 0")
	}
	logrus.Infof("master apply result: ToJoinNumber %d, ToDeleteIpList : %s",
		masterApplyResult.ToJoinNumber, masterApplyResult.ToDeleteIPList)

	logrus.Infof("node apply result: ToJoinNumber %d, ToDeleteIpList : %s",
		nodeApplyResult.ToJoinNumber, nodeApplyResult.ToDeleteIPList)

	if err := a.applyResult(masterApplyResult); err != nil {
		return err
	}
	return a.applyResult(nodeApplyResult)
}

func (a *ApplyProvider) applyResult(result *ApplyResult) error {
	// create or delete an update iplist
	switch result.Role {
	case MASTER:
		if result.ToJoinNumber > 0 {
			joinIPList, err := a.applyToJoin(result.ToJoinNumber, result.Role)
			if err != nil {
				return err
			}
			a.Cluster.Spec.Masters.IPList = append(a.Cluster.Spec.Masters.IPList, joinIPList...)
		}
		if len(result.ToDeleteIPList) > 0 {
			err := a.applyToDelete(result.ToDeleteIPList)
			if err != nil {
				return err
			}
			a.Cluster.Spec.Masters.IPList = a.Cluster.Spec.Masters.IPList[:len(a.Cluster.Spec.Masters.IPList)-
				len(result.ToDeleteIPList)]
		}
	case NODE:
		if result.ToJoinNumber > 0 {
			joinIPList, err := a.applyToJoin(result.ToJoinNumber, result.Role)
			if err != nil {
				return err
			}
			a.Cluster.Spec.Nodes.IPList = append(a.Cluster.Spec.Nodes.IPList, joinIPList...)
		}
		if len(result.ToDeleteIPList) > 0 {
			err := a.applyToDelete(result.ToDeleteIPList)
			if err != nil {
				return err
			}
			a.Cluster.Spec.Nodes.IPList = a.Cluster.Spec.Nodes.IPList[:len(a.Cluster.Spec.Nodes.IPList)-
				len(result.ToDeleteIPList)]
		}
	default:
		return fmt.Errorf("unknown node role: %q", result.Role)
	}
	return nil
}

func (a *ApplyProvider) applyToJoin(toJoinNumber int, role string) ([]net.IP, error) {
	// run container and return append ip list
	var toJoinIPList []net.IP
	for i := 0; i < toJoinNumber; i++ {
		name := fmt.Sprintf("sealer-%s-%s", role, GenUniqueID(10))
		opts := &client.CreateOptsForContainer{
			ImageName:         ImageName,
			NetworkName:       NetworkName,
			ContainerHostName: name,
			ContainerName:     name,
			ContainerLabel: map[string]string{
				RoleLabel: role,
			},
		}
		if len(a.Cluster.Spec.Masters.IPList) == 0 && i == 0 {
			opts.ContainerLabel[RoleLabelMaster] = "true"
		}

		containerID, err := a.Provider.RunContainer(opts)
		if err != nil {
			return toJoinIPList, fmt.Errorf("failed to create container %s,error is %v", opts.ContainerName, err)
		}
		time.Sleep(3 * time.Second)
		info, err := a.Provider.GetContainerInfo(containerID, NetworkName)
		if err != nil {
			return toJoinIPList, fmt.Errorf("failed to get container info of %s,error is %v", containerID, err)
		}

		err = a.changeDefaultPasswd(net.ParseIP(info.ContainerIP))
		if err != nil {
			return nil, fmt.Errorf("failed to change container password of %s,error is %v", containerID, err)
		}

		a.Cluster.Annotations[info.ContainerIP] = containerID
		toJoinIPList = append(toJoinIPList, net.ParseIP(info.ContainerIP))
	}
	return toJoinIPList, nil
}

func (a *ApplyProvider) changeDefaultPasswd(containerIP net.IP) error {
	if a.Cluster.Spec.SSH.Passwd == "" {
		return nil
	}

	if a.Cluster.Spec.SSH.Passwd == DefaultPassword {
		return nil
	}

	user := "root"
	if a.Cluster.Spec.SSH.User != "" {
		user = a.Cluster.Spec.SSH.User
	}
	sshClient := &ssh.SSH{
		User:     user,
		Password: DefaultPassword,
	}

	cmd := fmt.Sprintf(ChangePasswordCmd, a.Cluster.Spec.SSH.Passwd)
	_, err := sshClient.Cmd(containerIP, nil, cmd)
	return err
}

func (a *ApplyProvider) applyToDelete(deleteIPList []net.IP) error {
	// delete container and return deleted ip list
	for _, ip := range deleteIPList {
		id, ok := a.Cluster.Annotations[ip.String()]
		if !ok {
			logrus.Warnf("failed to delete container %s", ip)
			continue
		}
		err := a.Provider.RmContainer(id)
		if err != nil {
			return fmt.Errorf("failed to delete container:%s", id)
		}
		delete(a.Cluster.Annotations, ip.String())
	}
	return nil
}

func (a *ApplyProvider) CleanUp() error {
	//clean up container,cleanup image,clean up network
	var iplist []net.IP
	iplist = append(iplist, a.Cluster.Spec.Masters.IPList...)
	iplist = append(iplist, a.Cluster.Spec.Nodes.IPList...)

	for _, ip := range iplist {
		id, ok := a.Cluster.Annotations[ip.String()]
		if !ok {
			continue
		}
		err := a.Provider.RmContainer(id)
		if err != nil {
			// log it
			logrus.Infof("failed to delete container:%s", id)
			return err
		}
	}

	return nil
}

func NewClientWithCluster(cluster *v1.Cluster) (*ApplyProvider, error) {
	p, err := docker.NewDockerProvider()
	if err != nil {
		return nil, err
	}

	return &ApplyProvider{
		Cluster:  cluster,
		Provider: p,
	}, nil
}
