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

package common

import (
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

const (
	MASTER = "master"
	// TODO，警惕，不能通过此标志来获取worker，因为master也可以role=node
	NODE = "node"
)

// Default dir and file path
const (
	EtcDir               = "etc"
	KubeAdminConf        = "/etc/kubernetes/admin.conf"
	DefaultKubectlPath   = "/usr/bin/kubectl"
	DefaultTmpDir        = "/var/lib/sealer/tmp"
	DefaultLogDir        = "/var/lib/sealer/log"
	DefaultSealerDataDir = "/var/lib/sealer/data"
	ClusterfileName      = "ClusterfileName"
	RenderChartsDir      = "charts"
	RenderManifestsDir   = "manifests"
)

// API
const (
	APIVersion = "sealer.cloud/v2"
	Kind       = "Cluster"
)

// Envs
const (
	EnvHostIP           = "HostIP"
	EnvHostIPFamily     = "HostIPFamily"
	EnvIPv6DualStack    = "IPv6DualStack"
	EnvRegistryURL      = "RegistryURL"
	EnvRegistryDomain   = "RegistryDomain"
	EnvRegistryUsername = "RegistryUsername"
	EnvRegistryPassword = "RegistryPassword"
	EnvCertSANs         = "CertSANs"
)

const (
	ApplyModeApply     = "apply"
	ApplyModeLoadImage = "loadImage"
)

// image module
const (
	DefaultImageRootDir     = "/var/lib/sealer/data"
	DefaultMetadataName     = "Metadata"
	ImageScratch            = "scratch"
	DefaultImageMetaRootDir = "/var/lib/sealer/metadata"
	DefaultLayerDir         = "/var/lib/sealer/data/overlay2"
	DefaultRegistryDomain   = "sea.hub"
	DefaultRegistryPort     = "5000"
	DefaultRegistryURL      = DefaultRegistryDomain + ":" + DefaultRegistryPort
)

// about infra
const (
	AliDomain       = "sea.aliyun.com/"
	Eip             = AliDomain + "ClusterEIP"
	RegistryDirName = "registry"
)

// CRD kind
const (
	Config                 = "Config"
	Plugin                 = "Plugin"
	Cluster                = "Cluster"
	InitConfiguration      = "InitConfiguration"
	JoinConfiguration      = "JoinConfiguration"
	ClusterConfiguration   = "ClusterConfiguration"
	KubeletConfiguration   = "KubeletConfiguration"
	KubeProxyConfiguration = "KubeProxyConfiguration"
)

// plugin type
const (
	TAINT    = "TAINT"
	LABEL    = "LABEL"
	HOSTNAME = "HOSTNAME"
)

const (
	BAREMETAL = "BAREMETAL"
	AliCloud  = "ALI_CLOUD"
	CONTAINER = "CONTAINER"
)

const (
	FileMode0755 = 0755
	FileMode0644 = 0644
)

const APIServerDomain = "apiserver.cluster.local"

const (
	CdAndExecCmd        = "cd %s && %s"
	CdIfExistAndExecCmd = "if [ ! -d %s ];then exit 0;fi; cd %s && %s"
)

const (
	ExecBinaryFileName = "sealer"
	ROOT               = "root"
	WINDOWS            = "windows"
)

func GetSealerWorkDir() string {
	return filepath.Join(GetHomeDir(), ".sealer")
}

func GetDefaultClusterfile() string {
	return filepath.Join(GetSealerWorkDir(), "Clusterfile")
}

func GetDefaultApplicationFile() string {
	return filepath.Join(GetSealerWorkDir(), "application.json")
}

func DefaultRegistryAuthConfigDir() string {
	return filepath.Join(GetHomeDir(), ".docker/config.json")
}

func DefaultKubeConfigDir() string {
	return filepath.Join(GetHomeDir(), ".kube")
}

func DefaultKubeConfigFile() string {
	return filepath.Join(DefaultKubeConfigDir(), "config")
}

func DefaultTheClusterRootfsDir(clusterName string) string {
	return filepath.Join(DefaultSealerDataDir, clusterName, "rootfs")
}

func DefaultTheClusterNydusdDir(clusterName string) string {
	return filepath.Join(DefaultSealerDataDir, clusterName, "nydusd")
}

func DefaultTheClusterNydusdFileDir(clusterName string) string {
	return filepath.Join(DefaultSealerDataDir, clusterName, "nydusdfile")
}

func DefaultTheClusterRootfsPluginDir(clusterName string) string {
	return filepath.Join(DefaultTheClusterRootfsDir(clusterName), "plugins")
}

func TheDefaultClusterCertDir(clusterName string) string {
	return filepath.Join(DefaultSealerDataDir, clusterName, "certs")
}

func DefaultClusterBaseDir(clusterName string) string {
	return filepath.Join(DefaultSealerDataDir, clusterName)
}

func GetHomeDir() string {
	home, err := homedir.Dir()
	if err != nil {
		return "/root"
	}
	return home
}
