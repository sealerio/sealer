package upgrade

import "github.com/alibaba/sealer/utils/ssh"

type debian_distribution struct {
}

func (d debian_distribution) upgradeFirstMaster(client *ssh.Client, IP, version string) {

}
func (d debian_distribution) upgradeOtherMaster(client *ssh.Client, IP, version string) {

}
func (d debian_distribution) upgradeNode(client *ssh.Client, IP, version string) {

}
