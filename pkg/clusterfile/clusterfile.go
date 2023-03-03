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

package clusterfile

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/k8s"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utilsos "github.com/sealerio/sealer/utils/os"
)

const (
	configMapNamespace = "kube-system"
	configMapDataName  = "Clusterfile"
	configMapName      = "sealer-clusterfile"
)

type Interface interface {
	GetCluster() v2.Cluster
	SetCluster(v2.Cluster)
	GetConfigs() []v1.Config
	GetPlugins() []v1.Plugin
	GetApplication() *v2.Application
	GetKubeadmConfig() *kubeadm.KubeadmConfig
	SaveAll(opts SaveOptions) error
}

type SaveOptions struct {
	// if true ,will commit clusterfile to cluster
	CommitToCluster bool
	ConfPath        string
}

type ClusterFile struct {
	cluster       *v2.Cluster
	configs       []v1.Config
	kubeadmConfig kubeadm.KubeadmConfig
	plugins       []v1.Plugin
	apps          *v2.Application
}

func (c *ClusterFile) GetCluster() v2.Cluster {
	return *c.cluster
}

func (c *ClusterFile) SetCluster(cluster v2.Cluster) {
	c.cluster = &cluster
}

func (c *ClusterFile) GetConfigs() []v1.Config {
	return c.configs
}

func (c *ClusterFile) GetApplication() *v2.Application {
	return c.apps
}

func (c *ClusterFile) GetPlugins() []v1.Plugin {
	return c.plugins
}

func (c *ClusterFile) GetKubeadmConfig() *kubeadm.KubeadmConfig {
	return &c.kubeadmConfig
}

func (c *ClusterFile) SaveAll(opts SaveOptions) error {
	var (
		clusterfileBytes [][]byte
		config           []byte
		plugin           []byte
	)
	fileName := common.GetDefaultClusterfile()
	err := os.MkdirAll(filepath.Dir(fileName), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to mkdir %s: %v", fileName, err)
	}

	cluster, err := yaml.Marshal(c.cluster)
	if err != nil {
		return err
	}
	clusterfileBytes = append(clusterfileBytes, cluster)

	if len(c.configs) != 0 {
		for _, cg := range c.configs {
			config, err = yaml.Marshal(cg)
			if err != nil {
				return err
			}
			clusterfileBytes = append(clusterfileBytes, config)
		}
	}

	if len(c.plugins) != 0 {
		for _, p := range c.plugins {
			plugin, err = yaml.Marshal(p)
			if err != nil {
				return err
			}
			clusterfileBytes = append(clusterfileBytes, plugin)
		}
	}

	if len(c.kubeadmConfig.InitConfiguration.TypeMeta.Kind) != 0 {
		initConfiguration, err := yaml.Marshal(c.kubeadmConfig.InitConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, initConfiguration)
	}

	if len(c.kubeadmConfig.JoinConfiguration.TypeMeta.Kind) != 0 {
		joinConfiguration, err := yaml.Marshal(c.kubeadmConfig.JoinConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, joinConfiguration)
	}
	if len(c.kubeadmConfig.ClusterConfiguration.TypeMeta.Kind) != 0 {
		clusterConfiguration, err := yaml.Marshal(c.kubeadmConfig.ClusterConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, clusterConfiguration)
	}

	if len(c.kubeadmConfig.KubeletConfiguration.TypeMeta.Kind) != 0 {
		kubeletConfiguration, err := yaml.Marshal(c.kubeadmConfig.KubeletConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, kubeletConfiguration)
	}

	if len(c.kubeadmConfig.KubeProxyConfiguration.TypeMeta.Kind) != 0 {
		kubeProxyConfiguration, err := yaml.Marshal(c.kubeadmConfig.KubeProxyConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, kubeProxyConfiguration)
	}

	content := bytes.Join(clusterfileBytes, []byte("---\n"))
	err = utilsos.NewCommonWriter(fileName).WriteFile(content)
	if err != nil {
		return fmt.Errorf("failed to save clusterfile to disk:%v", err)
	}

	if opts.CommitToCluster {
		return saveToCluster(content, opts.ConfPath)
	}
	return nil
}

func saveToCluster(data []byte, confPath string) error {
	cli, err := kubernetes.GetClientFromConfig(confPath)
	if err != nil {
		return fmt.Errorf("failed to new k8s runtime client via adminconf: %v", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: configMapNamespace,
		},
		Data: map[string]string{configMapDataName: string(data)},
	}

	ctx := context.Background()
	if err := cli.Create(ctx, cm, &client.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create configmap: %v", err)
		}

		if err := cli.Update(ctx, cm, &client.UpdateOptions{}); err != nil {
			return fmt.Errorf("unable to update configmap: %v", err)
		}
	}

	return nil
}

func NewClusterFile(b []byte) (Interface, error) {
	clusterFile := new(ClusterFile)
	// use user specified clusterfile
	if len(b) > 0 {
		if err := decodeClusterFile(bytes.NewReader(b), clusterFile); err != nil {
			return nil, fmt.Errorf("failed to load clusterfile: %v", err)
		}
		return clusterFile, nil
	}

	// assume that we already have an existed cluster
	fromCluster, err := getClusterfileFromCluster()
	if err != nil {
		logrus.Warn("try to get clusterfile from cluster:", err)
	}
	if fromCluster != nil {
		return fromCluster, nil
	}
	// read local disk clusterfile
	workClusterfile := common.GetDefaultClusterfile()
	clusterFileData, err := os.ReadFile(filepath.Clean(workClusterfile))
	if err != nil {
		return nil, err
	}

	if err := decodeClusterFile(bytes.NewReader(clusterFileData), clusterFile); err != nil {
		return nil, fmt.Errorf("failed to load clusterfile: %v", err)
	}
	return clusterFile, nil
}

func getClusterfileFromCluster() (*ClusterFile, error) {
	clusterFile := new(ClusterFile)
	client, err := k8s.NewK8sClient()
	if err != nil {
		return nil, err
	}

	cm, err := client.ConfigMap(configMapNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	data := cm.Data[configMapDataName]
	if len(data) > 0 {
		err = decodeClusterFile(bytes.NewReader([]byte(data)), clusterFile)
		if err != nil {
			return nil, err
		}
		return clusterFile, nil
	}
	return nil, fmt.Errorf("failed to get clusterfile from cluster")
}
