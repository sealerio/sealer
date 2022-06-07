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

package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
)

const (
	SealerBinPath                     = "/usr/local/bin/sealer"
	ImageName                         = "sealer_test_image_"
	DefaultImageDomain                = "registry.cn-qingdao.aliyuncs.com"
	DefaultImageRepo                  = "sealer-io"
	DefaultImageName                  = "kubernetes:v1.19.8"
	DefaultRegistryAuthFileDir        = "/root/.docker"
	DefaultClusterFileNeedToBeCleaned = "/root/.sealer/%s/Clusterfile"
	SubCmdBuildOfSealer               = "build"
	SubCmdApplyOfSealer               = "apply"
	SubCmdDeleteOfSealer              = "delete"
	SubCmdRunOfSealer                 = "run"
	SubCmdLoginOfSealer               = "login"
	SubCmdTagOfSealer                 = "tag"
	SubCmdPullOfSealer                = "pull"
	SubCmdListOfSealer                = "images"
	SubCmdPushOfSealer                = "push"
	SubCmdRmiOfSealer                 = "rmi"
	DefaultSSHPassword                = "Sealer123"
	ImageAnnotationForClusterfile     = "sea.aliyun.com/ClusterFile"
)

const (
	FileMode0755 = 0755
	FileMode0644 = 0644
)
const (
	LiteBuild = "lite"
)
const (
	BAREMETAL         = "BAREMETAL"
	AliCloud          = "ALI_CLOUD"
	CONTAINER         = "CONTAINER"
	DefaultImage      = "registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8"
	DefaultNydusImage = "registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nydus:v1.19.8"
	ClusterNameForRun = "my-cluster"
	TMPClusterFile    = "/tmp/Clusterfile"
	ClusterWorkDir    = "/root/.sealer/%s"
)

var (
	DefaultPollingInterval time.Duration
	MaxWaiteTime           time.Duration
	DefaultWaiteTime       time.Duration
	DefaultSealerBin       = ""
	DefaultTestEnvDir      = ""
	RegistryURL            = os.Getenv("REGISTRY_URL")
	RegistryUsername       = os.Getenv("REGISTRY_USERNAME")
	RegistryPasswd         = os.Getenv("REGISTRY_PASSWORD")
	CustomImageName        = os.Getenv("IMAGE_NAME")
	CustomNydusImageName   = os.Getenv("NYDUS_IMAGE_NAME")

	AccessKey          = os.Getenv("ACCESSKEYID")
	AccessSecret       = os.Getenv("ACCESSKEYSECRET")
	Region             = os.Getenv("RegionID")
	TestImageName      = "" //default: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
	TestNydusImageName = "" //default: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nydus:v1.19.8
)

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
