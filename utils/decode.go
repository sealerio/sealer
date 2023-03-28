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

package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
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
)

var decodeCRDFuncMap = map[string]func(reader io.Reader) (interface{}, error){
	constants.ClusterKind:                     decodeClusterFunc,
	constants.ConfigKind:                      decodeConfigListFunc,
	constants.PluginKind:                      decodePluginListFunc,
	kubeadmConstants.InitConfigurationKind:    decodeInitConfigurationFunc,
	kubeadmConstants.JoinConfigurationKind:    decodeJoinConfigurationFunc,
	kubeadmConstants.ClusterConfigurationKind: decodeClusterConfigurationFunc,
	common.KubeletConfiguration:               decodeKubeletConfigurationFunc,
	common.KubeProxyConfiguration:             decodeKubeProxyConfigurationFunc,
}

// DecodeCRDFromFile decode custom resource definition from file, if not found, return io.EOF error.
func DecodeCRDFromFile(filepath string, kind string) (interface{}, error) {
	file, err := os.Open(path.Clean(filepath))
	if err != nil {
		return nil, fmt.Errorf("failed to open configfile(%s): %v", filepath, err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			logrus.Warnf("failed to dump config close clusterfile: %v", err)
		}
	}()
	return decodeCRDFuncMap[kind](file)
}

// DecodeCRDFromByte decode custom resource definition from byte slice, if not found, return io.EOF error.
func DecodeCRDFromByte(data []byte, kind string) (interface{}, error) {
	return decodeCRDFuncMap[kind](bytes.NewReader(data))
}

// DecodeCRDFromString decode custom resource definition from string, if not found, return io.EOF error.
func DecodeCRDFromString(data string, kind string) (interface{}, error) {
	return decodeCRDFuncMap[kind](strings.NewReader(data))
}

func NewK8sYamlDecoder(reader io.Reader) *yaml.YAMLToJSONDecoder {
	return yaml.NewYAMLToJSONDecoder(bufio.NewReaderSize(reader, 4096))
}

func decodeCRDFromReader(decoder *yaml.YAMLToJSONDecoder, kind string,
	unmarshalType func(version string) interface{} /*Get different constructs based on version and parse them.*/) (interface{}, error) {
	for {
		ext := runtime.RawExtension{}
		if err := decoder.Decode(&ext); err != nil {
			return nil, err
		}
		// TODO: This needs to be able to handle object in other encodings and schemas.
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		metaType := metav1.TypeMeta{}
		if err := yaml.Unmarshal(ext.Raw, &metaType); err != nil {
			return nil, fmt.Errorf("failed to decode cluster: %v", err)
		}
		if metaType.Kind != kind {
			continue
		}
		in := unmarshalType(metaType.APIVersion)
		if err := yaml.Unmarshal(ext.Raw, in); err != nil {
			return nil, fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
		}
		return in, nil
	}
}

func DecodeV1ClusterFromFile(filepath string) (*v1.Cluster, error) {
	file, err := os.Open(path.Clean(filepath))
	if err != nil {
		return nil, fmt.Errorf("failed to dump config: %v", err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			logrus.Warnf("failed to dump config close clusterfile: %v", err)
		}
	}()

	cluster, err := decodeCRDFromReader(NewK8sYamlDecoder(file), constants.ClusterKind, func(version string) interface{} { return &v1.Cluster{} })
	return cluster.(*v1.Cluster), err
}

func decodeClusterFunc(reader io.Reader) (out interface{}, err error) {
	switchVersion := func(version string) interface{} {
		switch version {
		case v1.GroupVersion.String():
			return &v1.Cluster{}
		case v2.GroupVersion.String():
			return &v2.Cluster{}
		default:
			return &v2.Cluster{}
		}
	}
	out, err = decodeCRDFromReader(NewK8sYamlDecoder(reader), constants.ClusterKind, switchVersion)
	if err != nil {
		return nil, err
	}
	//Compatible with v1
	if cluster, ok := out.(*v1.Cluster); ok {
		out = ConvertV1ClusterToV2Cluster(cluster)
	}
	return
}

func ConvertV1ClusterToV2Cluster(v1Cluster *v1.Cluster) *v2.Cluster {
	var (
		hosts   []v2.Host
		cluster = &v2.Cluster{}
	)
	if len(v1Cluster.Spec.Masters.IPList) != 0 {
		hosts = append(hosts, v2.Host{IPS: v1Cluster.Spec.Masters.IPList, Roles: []string{common.MASTER}})
	}
	if len(v1Cluster.Spec.Nodes.IPList) != 0 {
		hosts = append(hosts, v2.Host{IPS: v1Cluster.Spec.Nodes.IPList, Roles: []string{common.NODE}})
	}

	cluster.APIVersion = v2.GroupVersion.String()
	cluster.Spec.SSH = v1Cluster.Spec.SSH
	cluster.Spec.Env = v1Cluster.Spec.Env
	cluster.Spec.Hosts = hosts
	cluster.Spec.Image = v1Cluster.Spec.Image
	cluster.Name = v1Cluster.Name
	cluster.Kind = v1Cluster.Kind
	return cluster
}

func decodeConfigListFunc(reader io.Reader) (interface{}, error) {
	var (
		configs       []v1.Config
		decoder       = NewK8sYamlDecoder(reader)
		switchVersion = func(version string) interface{} { return &v1.Config{} }
	)
	for {
		in, err := decodeCRDFromReader(decoder, constants.ConfigKind, switchVersion)
		if err != nil {
			if err == io.EOF {
				return configs, nil
			}
			return nil, fmt.Errorf("failed to decode config: %v", err)
		}
		configs = append(configs, *in.(*v1.Config))
	}
}

func decodePluginListFunc(reader io.Reader) (interface{}, error) {
	var (
		plugins       []v1.Plugin
		decoder       = NewK8sYamlDecoder(reader)
		switchVersion = func(version string) interface{} { return &v1.Plugin{} }
	)

	for {
		in, err := decodeCRDFromReader(decoder, constants.PluginKind, switchVersion)
		if err != nil {
			if err == io.EOF {
				return plugins, nil
			}
			return nil, fmt.Errorf("failed to decode config: %v", err)
		}
		plugins = append(plugins, *in.(*v1.Plugin))
	}
}

func decodeInitConfigurationFunc(reader io.Reader) (out interface{}, err error) {
	switchVersion := func(version string) interface{} { return &v1beta3.InitConfiguration{} }
	return decodeCRDFromReader(NewK8sYamlDecoder(reader), kubeadmConstants.InitConfigurationKind, switchVersion)
}

func decodeJoinConfigurationFunc(reader io.Reader) (out interface{}, err error) {
	switchVersion := func(version string) interface{} { return &v1beta3.JoinConfiguration{} }
	return decodeCRDFromReader(NewK8sYamlDecoder(reader), kubeadmConstants.JoinConfigurationKind, switchVersion)
}

func decodeClusterConfigurationFunc(reader io.Reader) (out interface{}, err error) {
	switchVersion := func(version string) interface{} { return &v1beta3.ClusterConfiguration{} }
	return decodeCRDFromReader(NewK8sYamlDecoder(reader), kubeadmConstants.ClusterConfigurationKind, switchVersion)
}

func decodeKubeletConfigurationFunc(reader io.Reader) (out interface{}, err error) {
	switchVersion := func(version string) interface{} { return &v1beta1.KubeletConfiguration{} }
	return decodeCRDFromReader(NewK8sYamlDecoder(reader), common.KubeletConfiguration, switchVersion)
}

func decodeKubeProxyConfigurationFunc(reader io.Reader) (out interface{}, err error) {
	switchVersion := func(version string) interface{} { return &v1alpha1.KubeProxyConfiguration{} }
	return decodeCRDFromReader(NewK8sYamlDecoder(reader), common.KubeProxyConfiguration, switchVersion)
}
