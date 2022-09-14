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
	NODE    = "node"
	MASTER0 = "master0"
)

const (
	FROMCOMMAND        = "FROM"
	COPYCOMMAND        = "COPY"
	RUNCOMMAND         = "RUN"
	CMDCOMMAND         = "CMD"
	ENVCOMMAND         = "ENV"
	BaseImageLayerType = "BASE"
	RootfsLayerValue   = "rootfs cache"
)

const (
	DefaultWorkDir                = "/tmp/%s/workdir"
	EtcDir                        = "etc"
	DefaultTmpDir                 = "/var/lib/sealer/tmp"
	DefaultLiteBuildUpper         = "/var/lib/sealer/tmp/lite_build_upper"
	DefaultLogDir                 = "/var/lib/sealer/log"
	DefaultClusterFileName        = "Clusterfile"
	DefaultSealerDataDir          = "/var/lib/sealer/data"
	DefaultClusterInitBashFile    = "/var/lib/sealer/data/%s/scripts/init.sh"
	DefaultClusterCleanBashFile   = "/var/lib/sealer/data/%s/rootfs/scripts/clean.sh"
	TarGzSuffix                   = ".tar.gz"
	YamlSuffix                    = ".yaml"
	ImageAnnotationForClusterfile = "sea.aliyun.com/ClusterFile"
	RawClusterfile                = "/var/lib/sealer/Clusterfile"
	TmpClusterfile                = "/tmp/Clusterfile"
	DefaultRegistryHostName       = "registry.cn-qingdao.aliyuncs.com"
	DefaultRegistryAuthDir        = "/root/.docker/config.json"
	KubeAdminConf                 = "/etc/kubernetes/admin.conf"
	DefaultKubeDir                = "/root/.kube"
	DefaultKubectlPath            = "/usr/bin/kubectl"
	EtcHosts                      = "/etc/hosts"
	ClusterWorkDir                = "/root/.sealer/%s"
	RemoteSealerPath              = "/usr/local/bin/sealer"
	DefaultCloudProvider          = AliCloud
	ClusterfileName               = "ClusterfileName"
	CacheID                       = "cacheID"
	RenderChartsDir               = "charts"
	RenderManifestsDir            = "manifests"
	APIVersion                    = "sealer.cloud/v2"
	Kind                          = "Cluster"
	AppImage                      = "application"
)

// image module
const (
	DefaultImageRootDir     = "/var/lib/sealer/data"
	DefaultMetadataName     = "Metadata"
	ImageScratch            = "scratch"
	DefaultImageMetaRootDir = "/var/lib/sealer/metadata"
	DefaultLayerDir         = "/var/lib/sealer/data/overlay2"
)

//about infra
const (
	AliDomain         = "sea.aliyun.com/"
	Eip               = AliDomain + "ClusterEIP"
	RegistryDirName   = "registry"
	Master0InternalIP = AliDomain + "Master0InternalIP"
	EipID             = AliDomain + "EipID"
	Master0ID         = AliDomain + "Master0ID"
	VpcID             = AliDomain + "VpcID"
	VSwitchID         = AliDomain + "VSwitchID"
	SecurityGroupID   = AliDomain + "SecurityGroupID"
)

//CRD kind
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

//plugin type
const (
	TAINT    = "TAINT"
	LABEL    = "LABEL"
	HOSTNAME = "HOSTNAME"
)

const (
	JoinSubCmd   = "join"
	DeleteSubCmd = "delete"
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
	CdAndExecCmd = "cd %s && %s"
)

const (
	ExecBinaryFileName = "sealer"
	ROOT               = "root"
	WINDOWS            = "windows"
)

func GetClusterWorkDir() string {
	return filepath.Join(GetHomeDir(), ".sealer")
}

func GetClusterWorkClusterfile() string {
	return filepath.Join(GetClusterWorkDir(), "Clusterfile")
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
