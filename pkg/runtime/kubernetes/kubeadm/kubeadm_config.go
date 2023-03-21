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

package kubeadm

import (
	"fmt"
	"io"
	"net"
	"strings"

	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"

	versionUtils "github.com/sealerio/sealer/utils/version"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/utils"
	strUtils "github.com/sealerio/sealer/utils/strings"

	"github.com/imdario/mergo"
	"k8s.io/kube-proxy/config/v1alpha1"
	"k8s.io/kubelet/config/v1beta1"
)

// Read config from https://github.com/sealerio/sealer/blob/main/docs/design/clusterfile-v2.md and overwrite default kubeadm.yaml
// Use github.com/imdario/mergo to merge kubeadm config in Clusterfile and the default kubeadm config
// Using a config filter to handle some edge cases

// https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/apis/kubeadm/v1beta3/types.go
// Using map to overwrite Kubeadm configs

// nolint
type KubeadmConfig struct {
	v1beta3.InitConfiguration
	v1beta3.ClusterConfiguration
	v1alpha1.KubeProxyConfiguration
	v1beta1.KubeletConfiguration
	v1beta3.JoinConfiguration
}

const (
	EtcdServers = "etcd-servers"
)

const (
	V1991 = "v1.19.1"
	V1992 = "v1.19.2"
	V1150 = "v1.15.0"
	V1200 = "v1.20.0"
	V1230 = "v1.23.0"

	// kubeadm api version
	KubeadmV1beta1 = "kubeadm.k8s.io/v1beta1"
	KubeadmV1beta2 = "kubeadm.k8s.io/v1beta2"
	KubeadmV1beta3 = "kubeadm.k8s.io/v1beta3"
)

// LoadFromClusterfile :Load KubeadmConfig from Clusterfile.
// If it has `KubeadmConfig` in Clusterfile, load every field to each configuration.
// If Kubeadm raw config in Clusterfile, just load it.
func (k *KubeadmConfig) LoadFromClusterfile(kubeadmConfig KubeadmConfig) error {
	k.APIServer.CertSANs = strUtils.RemoveDuplicate(append(k.APIServer.CertSANs, kubeadmConfig.APIServer.CertSANs...))

	return mergo.Merge(k, kubeadmConfig)
}

// Merge Using github.com/imdario/mergo to merge KubeadmConfig to the ClusterImage default kubeadm config, overwrite some field.
// if defaultKubeadmConfig file not exist, use default raw kubeadm config to merge k.KubeConfigSpec empty value
func (k *KubeadmConfig) Merge(kubeadmYamlPath string, decode func(arg string, kind string) (interface{}, error)) error {
	newConfig, err := LoadKubeadmConfigs(kubeadmYamlPath, decode)
	if err != nil {
		return fmt.Errorf("failed to found kubeadm config from %s: %v", kubeadmYamlPath, err)
	}
	k.APIServer.CertSANs = strUtils.RemoveDuplicate(append(k.APIServer.CertSANs, newConfig.APIServer.CertSANs...))

	return mergo.Merge(k, newConfig)
}

func (k *KubeadmConfig) setAPIVersion(apiVersion string) {
	k.InitConfiguration.APIVersion = apiVersion
	k.ClusterConfiguration.APIVersion = apiVersion
	k.JoinConfiguration.APIVersion = apiVersion
}

func (k *KubeadmConfig) setKubeadmAPIVersion() {
	kv := versionUtils.Version(k.KubernetesVersion)
	greaterThanKV1150, err := kv.GreaterThan(V1150)
	if err != nil {
		logrus.Errorf("compare kubernetes version failed: %s", err)
	}
	greaterThanKV1230, err := kv.GreaterThan(V1230)
	if err != nil {
		logrus.Errorf("compare kubernetes version failed: %s", err)
	}
	switch {
	case greaterThanKV1150 && !greaterThanKV1230:
		k.setAPIVersion(KubeadmV1beta2)
	case greaterThanKV1230:
		k.setAPIVersion(KubeadmV1beta3)
	default:
		// Compatible with versions 1.14 and 1.13. but do not recommend.
		k.setAPIVersion(KubeadmV1beta1)
	}
}

func (k *KubeadmConfig) GetCertSANS() []string {
	return k.ClusterConfiguration.APIServer.CertSANs
}

func (k *KubeadmConfig) GetDNSDomain() string {
	return k.ClusterConfiguration.Networking.DNSDomain
}

func (k *KubeadmConfig) GetSvcCIDR() string {
	return k.ClusterConfiguration.Networking.ServiceSubnet
}

func LoadKubeadmConfigs(arg string, decode func(arg string, kind string) (interface{}, error)) (KubeadmConfig, error) {
	kubeadmConfig := KubeadmConfig{}
	initConfig, err := decode(arg, InitConfiguration)
	if err != nil && err != io.EOF {
		return kubeadmConfig, err
	} else if initConfig != nil {
		kubeadmConfig.InitConfiguration = *initConfig.(*v1beta3.InitConfiguration)
	}
	clusterConfig, err := decode(arg, ClusterConfiguration)
	if err != nil && err != io.EOF {
		return kubeadmConfig, err
	} else if clusterConfig != nil {
		kubeadmConfig.ClusterConfiguration = *clusterConfig.(*v1beta3.ClusterConfiguration)
	}
	kubeProxyConfig, err := decode(arg, KubeProxyConfiguration)
	if err != nil && err != io.EOF {
		return kubeadmConfig, err
	} else if kubeProxyConfig != nil {
		kubeadmConfig.KubeProxyConfiguration = *kubeProxyConfig.(*v1alpha1.KubeProxyConfiguration)
	}
	kubeletConfig, err := decode(arg, KubeletConfiguration)
	if err != nil && err != io.EOF {
		return kubeadmConfig, err
	} else if kubeletConfig != nil {
		kubeadmConfig.KubeletConfiguration = *kubeletConfig.(*v1beta1.KubeletConfiguration)
	}
	joinConfig, err := decode(arg, JoinConfiguration)
	if err != nil && err != io.EOF {
		return kubeadmConfig, err
	} else if joinConfig != nil {
		kubeadmConfig.JoinConfiguration = *joinConfig.(*v1beta3.JoinConfiguration)
	}
	return kubeadmConfig, nil
}

func getEtcdEndpointsWithHTTPSPrefix(masters []net.IP) string {
	var tmpSlice []string
	for _, ip := range masters {
		tmpSlice = append(tmpSlice, fmt.Sprintf("https://%s", net.JoinHostPort(ip.String(), "2379")))
	}

	return strings.Join(tmpSlice, ",")
}

func NewKubeadmConfig(fromClusterFile KubeadmConfig, fromFile string, masters []net.IP, apiServerDomain,
	cgroupDriver string, imageRepo string, apiServerVIP net.IP, extraSANs []string) (KubeadmConfig, error) {
	conf := KubeadmConfig{}

	if err := conf.LoadFromClusterfile(fromClusterFile); err != nil {
		return conf, fmt.Errorf("failed to load kubeadm config from clusterfile: %v", err)
	}
	// TODO handle the kubeadm config, like kubeproxy config
	//The configuration set here does not require merge

	conf.InitConfiguration.LocalAPIEndpoint.AdvertiseAddress = masters[0].String()
	conf.ControlPlaneEndpoint = net.JoinHostPort(apiServerDomain, "6443")

	if conf.APIServer.ExtraArgs == nil {
		conf.APIServer.ExtraArgs = make(map[string]string)
	}
	conf.APIServer.ExtraArgs[EtcdServers] = getEtcdEndpointsWithHTTPSPrefix(masters)
	conf.IPVS.ExcludeCIDRs = append(conf.KubeProxyConfiguration.IPVS.ExcludeCIDRs, fmt.Sprintf("%s/32", apiServerVIP))
	conf.KubeletConfiguration.CgroupDriver = cgroupDriver
	conf.ClusterConfiguration.APIServer.CertSANs = []string{"127.0.0.1", apiServerDomain, apiServerVIP.String()}
	conf.ClusterConfiguration.APIServer.CertSANs = append(conf.ClusterConfiguration.APIServer.CertSANs, extraSANs...)
	for _, m := range masters {
		conf.ClusterConfiguration.APIServer.CertSANs = append(conf.ClusterConfiguration.APIServer.CertSANs, m.String())
	}

	if err := conf.Merge(fromFile, utils.DecodeCRDFromFile); err != nil {
		return conf, err
	}

	if err := conf.Merge(DefaultKubeadmConfig, utils.DecodeCRDFromString); err != nil {
		return conf, err
	}

	conf.setKubeadmAPIVersion()

	if conf.ClusterConfiguration.Networking.DNSDomain == "" {
		conf.ClusterConfiguration.Networking.DNSDomain = "cluster.local"
	}
	if conf.JoinConfiguration.Discovery.BootstrapToken == nil {
		conf.JoinConfiguration.Discovery.BootstrapToken = &v1beta3.BootstrapTokenDiscovery{}
	}

	// set cluster image repo,kubeadm will pull container image from this registry.
	if conf.ClusterConfiguration.ImageRepository == "" {
		conf.ClusterConfiguration.ImageRepository = imageRepo
	}
	if conf.ClusterConfiguration.DNS.ImageMeta.ImageRepository == "" {
		conf.ClusterConfiguration.DNS.ImageMeta.ImageRepository = fmt.Sprintf("%s/%s", imageRepo, "coredns")
	}

	return conf, nil
}
