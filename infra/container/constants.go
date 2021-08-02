package container

const (
	DOCKER    = "docker"
	CONTAINER = "CONTAINER"
)
const (
	NETWROKID           = "NetworkId"
	IMAGEID             = "ImageId"
	DefaultPassword     = "Seadent123"
	ResourceNetwork     = "network"
	ResourceImage       = "image"
	DefaultNetworkName  = "sealer-network"
	DefaultImageName    = "registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-base-image:latest"
	DockerHost          = "/var/run/docker.sock"
	MASTER              = "master"
	NODE                = "node"
	SealerImageRootPath = "/var/lib/sealer"
	ChangePasswordCmd   = "echo root:%s | chpasswd"
	RoleLabel           = "sealer-io-role"
	RoleLabelMaster     = "sealer-io-role-is-master"
)
