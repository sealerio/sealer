// Copyright © 2022 Alibaba Group Holding Ltd.
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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kube-proxy/config/v1alpha1"
	"k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta3"
	kubeadmConstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/types/api/constants"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utilsnet "github.com/sealerio/sealer/utils/net"
	strUtil "github.com/sealerio/sealer/utils/strings"
)

func DecodeClusterfile(reader io.Reader) (*ClusterFile, error) {
	clusterFile := new(ClusterFile)
	// use user specified Clusterfile
	if err := decodeClusterFile(reader, clusterFile); err != nil {
		return nil, fmt.Errorf("failed to load clusterfile: %v", err)
	}
	return clusterFile, nil
}

func decodeClusterFile(reader io.Reader, clusterfile *ClusterFile) error {
	decoder := yaml.NewYAMLToJSONDecoder(bufio.NewReaderSize(reader, 4096))

	for {
		ext := runtime.RawExtension{}
		if err := decoder.Decode(&ext); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		metaType := metav1.TypeMeta{}
		if err := yaml.Unmarshal(ext.Raw, &metaType); err != nil {
			return fmt.Errorf("failed to decode TypeMeta: %v", err)
		}

		switch metaType.Kind {
		case constants.ClusterKind:
			var cluster v2.Cluster

			if err := yaml.Unmarshal(ext.Raw, &cluster); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}
			if err := checkAndFillCluster(&cluster); err != nil {
				return fmt.Errorf("failed to check and complete cluster: %v", err)
			}

			clusterfile.cluster = &cluster
		case constants.ConfigKind:
			var cfg v1.Config

			if err := yaml.Unmarshal(ext.Raw, &cfg); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			if cfg.Spec.Path == "" {
				return fmt.Errorf("failed to decode config %s, config path is empty", cfg.Name)
			}

			if cfg.Spec.Data == "" {
				return fmt.Errorf("failed to decode config %s, config data is empty", cfg.Name)
			}

			clusterfile.configs = append(clusterfile.configs, cfg)
		case constants.PluginKind:
			var plu v1.Plugin

			if err := yaml.Unmarshal(ext.Raw, &plu); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.plugins = append(clusterfile.plugins, plu)
		case constants.ApplicationKind:
			var app v2.Application

			if err := yaml.Unmarshal(ext.Raw, &app); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			for _, config := range app.Spec.Configs {
				if config.Name == "" {
					return fmt.Errorf("application configs name coule not be nil")
				}

				if config.Launch != nil {
					launchCmds := parseLaunchCmds(config.Launch)
					if launchCmds == nil {
						return fmt.Errorf("failed to get launchCmds from application configs")
					}
				}

				for _, appFile := range config.Files {
					if appFile.Data == "" {
						return fmt.Errorf("failed to decode application config %s. data is empty", config.Name)
					}

					if appFile.Path == "" {
						return fmt.Errorf("failed to decode application config %s. path is empty", config.Name)
					}
				}
			}

			clusterfile.app = &app
		case kubeadmConstants.InitConfigurationKind:
			var in v1beta3.InitConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.InitConfiguration = in
		case kubeadmConstants.JoinConfigurationKind:
			var in v1beta3.JoinConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.JoinConfiguration = in
		case kubeadmConstants.ClusterConfigurationKind:
			var in v1beta3.ClusterConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.ClusterConfiguration = in
		case common.KubeletConfiguration:
			var in v1beta1.KubeletConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.KubeletConfiguration = in
		case common.KubeProxyConfiguration:
			var in v1alpha1.KubeProxyConfiguration

			if err := yaml.Unmarshal(ext.Raw, &in); err != nil {
				return fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
			}

			clusterfile.kubeadmConfig.KubeProxyConfiguration = in
		}
	}
}

func checkAndFillCluster(cluster *v2.Cluster) error {
	defaultInsecure := false
	defaultHA := true

	if cluster.Spec.Registry.LocalRegistry == nil && cluster.Spec.Registry.ExternalRegistry == nil {
		cluster.Spec.Registry.LocalRegistry = &v2.LocalRegistry{}
	}

	if cluster.Spec.Registry.LocalRegistry != nil {
		if cluster.Spec.Registry.LocalRegistry.Domain == "" {
			cluster.Spec.Registry.LocalRegistry.Domain = common.DefaultRegistryDomain
		}
		if cluster.Spec.Registry.LocalRegistry.Port == 0 {
			cluster.Spec.Registry.LocalRegistry.Port = common.DefaultRegistryPort
		}
		if cluster.Spec.Registry.LocalRegistry.Insecure == nil {
			cluster.Spec.Registry.LocalRegistry.Insecure = &defaultInsecure
		}
		if cluster.Spec.Registry.LocalRegistry.HA == nil {
			cluster.Spec.Registry.LocalRegistry.HA = &defaultHA
		}
	}

	if cluster.Spec.Registry.ExternalRegistry != nil {
		if cluster.Spec.Registry.ExternalRegistry.Domain == "" {
			return fmt.Errorf("external registry domain can not be empty")
		}
	}

	var newEnv []string
	for _, env := range cluster.Spec.Env {
		if strings.HasPrefix(env, common.EnvLocalRegistryDomain) ||
			strings.HasPrefix(env, common.EnvLocalRegistryPort) ||
			strings.HasPrefix(env, common.EnvLocalRegistryURL) ||
			strings.HasPrefix(env, common.EnvExternalRegistryDomain) ||
			strings.HasPrefix(env, common.EnvExternalRegistryPort) ||
			strings.HasPrefix(env, common.EnvExternalRegistryURL) ||
			strings.HasPrefix(env, common.EnvRegistryDomain) ||
			strings.HasPrefix(env, common.EnvRegistryPort) ||
			strings.HasPrefix(env, common.EnvRegistryURL) ||
			strings.HasPrefix(env, common.EnvContainerRuntime) ||
			strings.HasPrefix(env, common.EnvDNSSvcIP) ||
			strings.HasPrefix(env, common.EnvKubeSvcIP) {
			continue
		}
		newEnv = append(newEnv, env)
	}
	cluster.Spec.Env = newEnv

	clusterEnvMap := strUtil.ConvertStringSliceToMap(cluster.Spec.Env)
	if svcCIDR, ok := clusterEnvMap[common.EnvSvcCIDR]; ok && svcCIDR != "" {
		cidrs := strings.Split(svcCIDR, ",")
		_, cidr, err := net.ParseCIDR(cidrs[0])
		if err != nil {
			return fmt.Errorf("failed to parse svc CIDR: %v", err)
		}
		kubeIP, err := utilsnet.GetIndexIP(cidr, 1)
		if err != nil {
			return fmt.Errorf("failed to get 1th ip from svc CIDR: %v", err)
		}
		dnsIP, err := utilsnet.GetIndexIP(cidr, 10)
		if err != nil {
			return fmt.Errorf("failed to get 10th ip from svc CIDR: %v", err)
		}
		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvKubeSvcIP, kubeIP))
		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvDNSSvcIP, dnsIP))
	}

	regConfig := v2.RegistryConfig{}
	if cluster.Spec.Registry.LocalRegistry != nil {
		regConfig = cluster.Spec.Registry.LocalRegistry.RegistryConfig

		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvLocalRegistryDomain, regConfig.Domain))
		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%d", common.EnvLocalRegistryPort, regConfig.Port))
		registryURL := net.JoinHostPort(regConfig.Domain, strconv.Itoa(regConfig.Port))
		if regConfig.Port == 0 {
			registryURL = regConfig.Domain
		}
		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvLocalRegistryURL, registryURL))
	}
	if cluster.Spec.Registry.ExternalRegistry != nil {
		regConfig = cluster.Spec.Registry.ExternalRegistry.RegistryConfig

		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvExternalRegistryDomain, regConfig.Domain))
		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%d", common.EnvExternalRegistryPort, regConfig.Port))
		registryURL := net.JoinHostPort(regConfig.Domain, strconv.Itoa(regConfig.Port))
		if regConfig.Port == 0 {
			registryURL = regConfig.Domain
		}
		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvExternalRegistryURL, registryURL))
	}

	cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvRegistryDomain, regConfig.Domain))
	portStr := fmt.Sprintf("%d", regConfig.Port)
	if regConfig.Port == 0 {
		portStr = ""
	}
	cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvRegistryPort, portStr))
	registryURL := net.JoinHostPort(regConfig.Domain, strconv.Itoa(regConfig.Port))
	if regConfig.Port == 0 {
		registryURL = regConfig.Domain
	}
	cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvRegistryURL, registryURL))

	if cluster.Spec.ContainerRuntime.Type != "" {
		cluster.Spec.Env = append(cluster.Spec.Env, fmt.Sprintf("%s=%s", common.EnvContainerRuntime, cluster.Spec.ContainerRuntime.Type))
	}

	if cluster.Spec.DataRoot == "" {
		cluster.Spec.DataRoot = common.DefaultSealerDataDir
	}

	return nil
}

// parseLaunchCmds parse shell, kube,helm type launch cmds
// kubectl apply -n sealer-io -f ns.yaml -f app.yaml
// helm install my-nginx bitnami/nginx
// key1=value1 key2=value2 && bash install1.sh && bash install2.sh
func parseLaunchCmds(launch *v2.Launch) []string {
	if launch.Cmds != nil {
		return launch.Cmds
	}
	// TODO add shell,helm,kube type cmds.
	return nil
}
