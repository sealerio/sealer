package container

const (
	DOCKER             = "docker"
	CONTAINER          = "CONTAINER"
	BUILD_ARG_PASSWORD = "PASSWORD"
)
const (
	NETWROK_ID             = "NetworkId"
	IMAGE_ID               = "ImageId"
	DEFAULT_PASSWORD       = "Seadent123"
	RESOURCE_NETWORK       = "network"
	RESOURCE_IMAGE         = "image"
	DEFAULT_NETWORK_NAME   = "sealer-network"
	DEFAULT_IMAGE_NAME     = "registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-base-image:latest"
	DOCKER_HOST            = "/var/run/docker.sock"
	MASTER                 = "master"
	NODE                   = "node"
	SEALER_IMAGE_ROOT_PATH = "/var/lib/sealer"
	CHANGE_PASSWORD_CMD    = "echo root:%s | chpasswd"

	CONTAINERLABLE       = "sealer-io-role"
	CONTAINERLABLEMASTER = "sealer-io-role-is-master"
)
