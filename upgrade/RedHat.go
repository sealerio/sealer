package upgrade

import "github.com/alibaba/sealer/utils/ssh"

type redhat_distribution struct {
}

func (r redhat_distribution) upgradeFirstMaster(client *ssh.Client, IP, version string) {

}
func (r redhat_distribution) upgradeOtherMaster(client *ssh.Client, IP, version string) {

}
func (r redhat_distribution) upgradeNode(client *ssh.Client, IP, version string) {

}
