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
	"net"
	"strconv"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/filesystem"
	"github.com/sealerio/sealer/pkg/filesystem/clusterimage"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utilsnet "github.com/sealerio/sealer/utils/net"
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
	Runtime      *kubernetes.Runtime
	ImageEngine  imageengine.Interface
	ImageMounter clusterimage.Interface
}

func NewGenerateProcessor() (Processor, error) {
	imageEngine, err := imageengine.NewImageEngine(options.EngineGlobalConfigurations{})
	if err != nil {
		return nil, err
	}

	imageMounter, err := filesystem.NewClusterImageMounter(imageEngine)
	if err != nil {
		return nil, err
	}

	return &GenerateProcessor{
		ImageEngine:  imageEngine,
		ImageMounter: imageMounter,
	}, nil
}

func (g *GenerateProcessor) init(cluster *v2.Cluster) error {
	fileName := fmt.Sprintf("%s/.sealer/%s/Clusterfile", common.GetHomeDir(), cluster.Name)
	if err := clusterfile.SaveToDisk(cluster, fileName); err != nil {
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
	var nodeip, masterip []net.IP

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
		return nil, fmt.Errorf("failed to create k8s client: %s", err)
	}

	all, err := c.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %s", err)
	}
	for _, n := range all.Items {
		for _, v := range n.Status.Addresses {
			if _, ok := n.Labels[masterLabel]; ok {
				if v.Type == v1.NodeInternalIP {
					masterip = append(masterip, net.ParseIP(v.Address))
				}
			} else if v.Type == v1.NodeInternalIP {
				nodeip = append(nodeip, net.ParseIP(v.Address))
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
	hosts := cluster.GetAllIPList()
	regConfig := registry.GetConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.GetMaster0IP())
	if utilsnet.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return fs.MountRootfs(cluster, hosts, false)
}

func (g *GenerateProcessor) MountImage(cluster *v2.Cluster) error {
	platsMap, err := ssh.GetClusterPlatform(cluster)
	if err != nil {
		return err
	}

	tmpMap := map[string]struct{}{}
	tmpMap[platform.GetDefaultPlatform().ToString()] = struct{}{}
	platforms := []string{platform.GetDefaultPlatform().ToString()}
	for _, v := range platsMap {
		if _, ok := tmpMap[v.ToString()]; ok {
			continue
		}
		platforms = append(platforms, v.ToString())
	}

	image := cluster.Spec.Image
	for _, p := range platforms {
		if err := g.ImageEngine.Pull(&options.PullOptions{
			PullPolicy: "missing",
			Image:      image,
			Platform:   p,
		}); err != nil {
			return fmt.Errorf("failed to pull image %s with platform %s: %v", image, p, err)
		}
	}

	if err = g.ImageMounter.MountImage(cluster); err != nil {
		return err
	}
	runt, err := ChooseRuntime(platform.DefaultMountClusterImageDir(cluster.Name), cluster, nil)
	if err != nil {
		return err
	}
	g.Runtime = runt.(*kubernetes.Runtime)
	return nil
}

func (g *GenerateProcessor) UnmountImage(cluster *v2.Cluster) error {
	return g.ImageMounter.UnMountImage(cluster)
}

func (g *GenerateProcessor) ApplyRegistry(cluster *v2.Cluster) error {
	runt, err := ChooseRuntime(platform.DefaultMountClusterImageDir(cluster.Name), cluster, nil)
	if err != nil {
		return err
	}
	rt, ok := runt.(*kubernetes.Runtime)
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
