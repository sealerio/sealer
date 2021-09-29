package upgrade

type Interface interface {
	upgrade(version string)
}

func UpgradeCluster(version string) error {
	//TODO 判断是否是一个可升级的版本（有这个版本，且是新版本）

	//获取当前集群的各个节点的IP地址和ssh密码（root用户）

	//登录到每个节点上，调用对应linux发行版的升级函数
	return nil
}
