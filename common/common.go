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
	DefaultImageRootDir           = "/var/lib/seadent/data"
	DefaultWorkDir                = "/var/lib/seadent/%s/workdir"
	DefaultClusterFileName        = "Clusterfile"
	DefaultClusterRootfsDir       = "/var/lib/seadent/data"
	DefaultClusterInitFile        = "/var/lib/seadent/data/%s/scripts/init.sh"
	DefaultClusterClearFile       = "/var/lib/seadent/data/%s/scripts/clean.sh"
	ImageScratch                  = "scratch"
	DefaultImageMetaRootDir       = "/var/lib/seadent/metadata"
	DefaultImageMetadataFile      = "/var/lib/seadent/metadata/images_metadata.json"
	DefaultLayerDir               = "/var/lib/seadent/data/overlay2"
	YamlSuffix                    = ".yaml"
	RemoteServerEIPAnnotation     = "sea.aliyun.com/ClusterEIP"
	ImageAnnotationForClusterfile = "sea.aliyun.com/ClusterFile"
	RawClusterfile                = "/var/lib/seadent/Clusterfile"
	TmpClusterfile                = "/tmp/Clusterfile"
	DefaultRegistryHostName       = "registry.cn-qingdao.aliyuncs.com"
	DefaultRegistryAuthDir        = "/root/.docker/config.json"
	KubeAdminConf                 = "/etc/kubernetes/admin.conf"
	DefaultKubeconfig             = "/root/.kube/config"
	DefaultKubeconfigDir          = "/root/.kube"
	KubectlPath                   = "/usr/bin/kubectl"
	EtcHosts                      = "/etc/hosts"
	ClusterWorkDir                = "/root/.seadent/%s"
	ClusterWorkClusterfile        = "/root/.seadent/%s/Clusterfile"
	RemoteSealerPath              = "/usr/local/bin/sealer"
)

//about infra
const (
	AliDomain         = "sea.aliyun.com/"
	Eip               = AliDomain + "ClusterEIP"
	Master0InternalIP = AliDomain + "Master0InternalIP"
)

const (
	LocalBuild = "local"
)
const (
	BAREMETAL = "BAREMETAL"
	ALI_CLOUD = "ALI_CLOUD"
)

const (
	FileMode0766 = 0766
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
	BuildClusterCmd = "%s build -f %s -t %s -b %s ."
)
const ExecBinaryFileName = "sealer"
const ROOT = "root"

func GetClusterWorkDir(clusterName string) string {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Sprintf(ClusterWorkDir, clusterName)
	}
	return filepath.Join(home, ".seadent", clusterName)
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
