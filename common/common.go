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
	"fmt"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

const (
	FROMCOMMAND = "FROM"
	COPYCOMMAND = "COPY"
	RUNCOMMAND  = "RUN"
	CMDCOMMAND  = "CMD"
	ENVCOMMAND  = "ENV"
)

const (
	DefaultWorkDir                = "/var/lib/sealer/%s/workdir"
	DefaultClusterFileName        = "Clusterfile"
	DefaultClusterRootfsDir       = "/var/lib/sealer/data"
	DefaultClusterInitFile        = "/var/lib/sealer/data/%s/scripts/init.sh"
	DefaultClusterClearFile       = "/var/lib/sealer/data/%s/rootfs/scripts/clean.sh"
	TarGzSuffix                   = ".tar.gz"
	YamlSuffix                    = ".yaml"
	ImageAnnotationForClusterfile = "sea.aliyun.com/ClusterFile"
	RawClusterfile                = "/var/lib/sealer/Clusterfile"
	TmpClusterfile                = "/tmp/Clusterfile"
	DefaultRegistryHostName       = "registry.cn-qingdao.aliyuncs.com"
	DefaultRegistryAuthDir        = "/root/.docker/config.json"
	KubeAdminConf                 = "/etc/kubernetes/admin.conf"
	DefaultKubeDir                = "/root/.kube"
	KubectlPath                   = "/usr/bin/kubectl"
	EtcHosts                      = "/etc/hosts"
	ClusterWorkDir                = "/root/.sealer/%s"
	RemoteSealerPath              = "/usr/local/bin/sealer"
	DefaultCloudProvider          = AliCloud
	ClusterfileName               = "ClusterfileName"
)

// image module
const (
	DefaultImageRootDir          = "/var/lib/sealer/data"
	DefaultMetadataName          = "Metadata"
	DefaultImageMetadataFileName = "image_metadata.yaml"
	ImageScratch                 = "scratch"
	DefaultImageMetaRootDir      = "/var/lib/sealer/metadata"
	DefaultImageDBRootDir        = "/var/lib/sealer/metadata/imagedb"
	DefaultImageMetadataFile     = "/var/lib/sealer/metadata/images_metadata.json"
	DefaultLayerDir              = "/var/lib/sealer/data/overlay2"
	DefaultLayerDBRoot           = "/var/lib/sealer/metadata/layerdb"
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
	CRDConfig = "Config"
)

const (
	LocalBuild = "local"
)
const (
	BAREMETAL = "BAREMETAL"
	AliCloud  = "ALI_CLOUD"
)

const (
	FileMode0755 = 0755
	FileMode0644 = 0644
)

const APIServerDomain = "apiserver.cluster.local"

const (
	DeleteCmd       = "rm -rf %s"
	ChmodCmd        = "chmod +x %s"
	TmpTarFile      = "/tmp/%s.tar.gz"
	ZipCmd          = "tar zcvf %s %s"
	UnzipCmd        = "mkdir -p %s && tar zxvf %s -C %s"
	CdAndExecCmd    = "cd %s && %s"
	TagImageCmd     = "%s tag %s %s"
	PushImageCmd    = "%s push %s"
	BuildClusterCmd = "%s build -f %s -t %s -b %s %s"
)
const ExecBinaryFileName = "sealer"
const ROOT = "root"
const WINDOWS = "windows"

func GetClusterWorkDir(clusterName string) string {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Sprintf(ClusterWorkDir, clusterName)
	}
	return filepath.Join(home, ".sealer", clusterName)
}

func GetClusterWorkClusterfile(clusterName string) string {
	return filepath.Join(GetClusterWorkDir(clusterName), "Clusterfile")
}

func DefaultRegistryAuthConfigDir() string {
	dir, err := homedir.Dir()
	if err != nil {
		return DefaultRegistryAuthDir
	}

	return filepath.Join(dir, ".docker/config.json")
}

func DefaultKubeConfigDir() string {
	home, err := homedir.Dir()
	if err != nil {
		return DefaultKubeDir
	}
	return filepath.Join(home, ".kube")
}

func DefaultKubeConfigFile() string {
	return filepath.Join(DefaultKubeConfigDir(), "config")
}

func DefaultMountCloudImageDir(clusterName string) string {
	return filepath.Join(DefaultClusterRootfsDir, clusterName, "mount")
}

func DefaultTheClusterRootfsDir(clusterName string) string {
	return filepath.Join(DefaultClusterRootfsDir, clusterName, "rootfs")
}

func DefaultClusterBaseDir(clusterName string) string {
	return filepath.Join(DefaultClusterRootfsDir, clusterName)
}
