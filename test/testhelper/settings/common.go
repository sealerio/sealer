package settings

import (
	"os"
	"time"
)

const (
	SealerBinPath                     = "/usr/local/bin/sealer"
	ImageName                         = "sealer_test_image_"
	DefaultRegistryAuthDir            = "/root/.docker"
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
)

const (
	LocalBuild = "local"
)
const (
	BAREMETAL = "BAREMETAL"
	AliCloud  = "ALI_CLOUD"
)

var (
	DefaultPollingInterval time.Duration
	MaxWaiteTime           time.Duration
	DefaultWaiteTime       time.Duration

	RegistryURL      = os.Getenv("REGISTRY_URL")
	RegistryUsername = os.Getenv("REGISTRY_USERNAME")
	RegistryPasswd   = os.Getenv("REGISTRY_PASSWORD")

	AccessKey     = os.Getenv("ACCESSKEYID")
	AccessSecret  = os.Getenv("ACCESSKEYSECRET")
	Region        = os.Getenv("RegionID")
	TestImageName = ""
)
