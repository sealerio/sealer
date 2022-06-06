/*
Copyright © 2022 Alibaba Group Holding Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package processor

import (
	"fmt"
	"strconv"

	"github.com/sealerio/sealer/utils/yaml"

	"github.com/sealerio/sealer/utils/net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
	"github.com/sealerio/sealer/pkg/filesystem"
	"github.com/sealerio/sealer/pkg/filesystem/cloudimage"
	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/pkg/runtime"
	apiv1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"

	v1 "k8s.io/api/core/v1"
)

const (
	masterLabel = "node-role.kubernetes.io/master"
)

type ParserArg struct {
	Name       string
	Passwd     string
	Image      string
	Port       uint16
	Pk         string
	PkPassword string
}

type GenerateProcessor struct {
	Runtime      *runtime.KubeadmRuntime
	ImageManager image.Service
	ImageMounter cloudimage.Interface
}

func NewGenerateProcessor() (Processor, error) {
	imageMounter, err := filesystem.NewCloudImageMounter()
	if err != nil {
		return nil, err
	}
	imgSvc, err := image.NewImageService()
	if err != nil {
		return nil, err
	}
	return &GenerateProcessor{
		ImageManager: imgSvc,
		ImageMounter: imageMounter,
	}, nil
}

func (g *GenerateProcessor) init(cluster *v2.Cluster) error {
	fileName := fmt.Sprintf("%s/.sealer/%s/Clusterfile", common.GetHomeDir(), cluster.Name)
	if err := yaml.MarshalToFile(fileName, cluster); err != nil {
		return err
	}
	return nil
}

func (g *GenerateProcessor) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	todoList = append(todoList,
		g.init,
		g.MountImage,
		g.MountRootfs,
		g.ApplyRegistry,
		g.UnmountImage,
	)
	return todoList, nil
}

func GenerateCluster(arg *ParserArg) (*v2.Cluster, error) {
	var nodeip, masterip []string
	cluster := &v2.Cluster{}

	cluster.Kind = common.Kind
	cluster.APIVersion = common.APIVersion
	cluster.Name = arg.Name
	cluster.Spec.Image = arg.Image
	cluster.Spec.SSH.Passwd = arg.Passwd
	cluster.Spec.SSH.Port = strconv.Itoa(int(arg.Port))
	cluster.Spec.SSH.Pk = arg.Pk
	cluster.Spec.SSH.PkPasswd = arg.PkPassword

	c, err := k8s.Newk8sClient()
	if err != nil {
		return nil, fmt.Errorf("generate clusterfile failed, %s", err)
	}

	all, err := c.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("generate clusterfile failed, %s", err)
	}
	for _, n := range all.Items {
		for _, v := range n.Status.Addresses {
			if _, ok := n.Labels[masterLabel]; ok {
				if v.Type == v1.NodeInternalIP {
					masterip = append(masterip, v.Address)
				}
			} else if v.Type == v1.NodeInternalIP {
				nodeip = append(nodeip, v.Address)
			}
		}
	}

	masterHosts := v2.Host{
		IPS:   masterip,
		Roles: []string{common.MASTER},
	}

	nodeHosts := v2.Host{
		IPS:   nodeip,
		Roles: []string{common.NODE},
	}

	cluster.Spec.Hosts = append(cluster.Spec.Hosts, masterHosts, nodeHosts)
	return cluster, nil
}

func (g *GenerateProcessor) MountRootfs(cluster *v2.Cluster) error {
	fs, err := filesystem.NewFilesystem(common.DefaultTheClusterRootfsDir(cluster.Name))
	if err != nil {
		return err
	}
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	regConfig := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.GetMaster0IP())
	if net.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return fs.MountRootfs(cluster, hosts, false)
}

func (g *GenerateProcessor) MountImage(cluster *v2.Cluster) error {
	platsMap, err := ssh.GetClusterPlatform(cluster)
	if err != nil {
		return err
	}
	plats := []*apiv1.Platform{platform.GetDefaultPlatform()}
	for _, v := range platsMap {
		plat := v
		plats = append(plats, &plat)
	}
	err = g.ImageManager.PullIfNotExist(cluster.Spec.Image, plats)
	if err != nil {
		return err
	}
	if err = g.ImageMounter.MountImage(cluster); err != nil {
		return err
	}
	runt, err := runtime.NewDefaultRuntime(cluster, nil)
	if err != nil {
		return err
	}
	g.Runtime = runt.(*runtime.KubeadmRuntime)
	return nil
}

func (g *GenerateProcessor) UnmountImage(cluster *v2.Cluster) error {
	return g.ImageMounter.UnMountImage(cluster)
}

func (g *GenerateProcessor) ApplyRegistry(cluster *v2.Cluster) error {
	runt, err := runtime.NewDefaultRuntime(cluster, nil)
	if err != nil {
		return err
	}
	rt, ok := runt.(*runtime.KubeadmRuntime)
	if !ok {
		return fmt.Errorf("invalid type")
	}
	err = rt.GenerateRegistryCert()
	if err != nil {
		return err
	}
	err = rt.SendRegistryCert(cluster.GetAllIPList())
	if err != nil {
		return err
	}
	return g.Runtime.ApplyRegistry()
}
