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

package guest

import (
	"context"
	"time"

	"github.com/alibaba/sealer/client/k8s"
	"github.com/alibaba/sealer/logger"

	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	ssh2 "github.com/alibaba/sealer/utils/ssh"
)

type Interface interface {
	Apply(cluster *v1.Cluster) error
	Delete(cluster *v1.Cluster) error
}

type Default struct {
	imageStore store.ImageStore
}

func NewGuestManager() (Interface, error) {
	is, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return &Default{imageStore: is}, nil
}

func (d *Default) Apply(cluster *v1.Cluster) error {
	ssh := ssh2.NewSSHByCluster(cluster)
	image, err := d.imageStore.GetByName(cluster.Spec.Image)
	if err != nil {
		return fmt.Errorf("get cluster image failed, %s", err)
	}
	masters := cluster.Spec.Masters.IPList
	if len(masters) == 0 {
		return fmt.Errorf("failed to found master")
	}
	clusterRootfs := common.DefaultTheClusterRootfsDir(cluster.Name)
	imageCMDLayers := make([]v1.Layer, 0)
	for i := range image.Spec.Layers {
		if image.Spec.Layers[i].Type == common.CMDCOMMAND {
			imageCMDLayers = append(imageCMDLayers, image.Spec.Layers[i])
		}
	}
	for i := range imageCMDLayers[0:2] {
		if err := ssh.CmdAsync(masters[0], fmt.Sprintf(common.CdAndExecCmd, clusterRootfs, imageCMDLayers[i].Value)); err != nil {
			return err
		}
	}
	if err := d.waitClusterReady(context.TODO()); err != nil {
		return err
	}
	for i := range imageCMDLayers[2:] {
		if err := ssh.CmdAsync(masters[0], fmt.Sprintf(common.CdAndExecCmd, clusterRootfs, imageCMDLayers[i+2].Value)); err != nil {
			return err
		}
	}
	return nil
}

func (d Default) Delete(cluster *v1.Cluster) error {
	panic("implement me")
}

func (d *Default) waitClusterReady(ctx context.Context) error {
	var clusterStatusChan = make(chan string)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func(t *time.Ticker) {
		for {
			clusterStatus := d.getClusterStatus()
			clusterStatusChan <- clusterStatus
			<-t.C
		}
	}(ticker)
	for {
		select {
		case status := <-clusterStatusChan:
			if status == common.ClusterNotReady {
				logger.Info("wait cluster ready ")
			} else if status == common.ClusterReady {
				logger.Info("cluster ready now")
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("cluster is not ready, please check")
		}
	}
}

func (d *Default) getClusterStatus() string {
	k8sClient, err := k8s.Newk8sClient()
	if err != nil {
		return common.ClusterNotReady
	}

	podStatusList, err := k8sClient.ListAllNamespacesPodsStatus()
	if podStatusList == nil || err != nil {
		return common.ClusterNotReady
	}

	for _, status := range podStatusList {
		if !status {
			return common.ClusterNotReady
		}
	}
	return common.ClusterReady
}
