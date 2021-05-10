package runtime

import (
	"fmt"
	"os"
	"time"

	"github.com/alibaba/sealer/client"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
)

func (d *Default) upgrade() error {
	err := d.UpgradeMasters0()
	if err != nil {
		return err
	}

	if len(d.Masters) > 1 {
		err = d.UpgradeOtherMasters()
	}

	if len(d.Nodes) >= 1 {
		err = d.UpgradeNodes()
	}

	return err
}

func (d *Default) UpgradeMasters0() error {
	logger.Info("Upgrade Masters0, %s", d.Masters[:1])
	return d.upgradeNodes(d.Masters[:1], true)
}

func (d *Default) UpgradeOtherMasters() error {
	logger.Info("UpgradeMasters")
	return d.upgradeNodes(d.Masters[1:], true)
}

func (d *Default) UpgradeNodes() error {
	logger.Info("UpgradeNodes")
	return d.upgradeNodes(d.Nodes, false)
}

func (d *Default) upgradeNodes(hosts []string, isMaster bool) error {
	wg := NewPool(2)
	var err error
	c, err := client.NewClientSet()
	if err != nil {
		logger.Info("current cluster not found, upgrade nothing %v", err)
		return err
	}
	var errList []error
	for _, host := range hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			node := d.GetRemoteHostName(h)
			ip := utils.GetHostIP(h)
			// drain worker node is too danger for prod use; do not drain nodes if worker nodes~
			if isMaster {
				logger.Info("[%s] first: to drain master node %s", ip, node)
				cmdDrain := fmt.Sprintf(`kubectl drain %s --ignore-daemonsets --delete-local-data`, node)
				err := d.SSH.CmdAsync(d.Masters[0], cmdDrain)
				if err != nil {
					logger.Error("kubectl drain %s  err: %v", node, err)
					errList = append(errList, err)
				}
			} else {
				logger.Info("first: to print upgrade node %s", node)
			}

			logger.Info("first: to print upgrade node %s", node)

			// second to exec kubeadm upgrade node
			logger.Info("[%s] second: to exec kubeadm upgrade node on %s", ip, node)
			var cmdUpgrade string
			if ip == d.Masters[0] {
				cmdUpgrade = fmt.Sprintf("kubeadm upgrade apply --certificate-renewal=false  --yes %s", d.Version)
				err = d.SSH.CmdAsync(ip, cmdUpgrade)
				if err != nil {
					// master1 upgrade failed exit.
					logger.Error("kubeadm upgrade err: ", err)
					errList = append(errList, err)
					os.Exit(1)
				}
			} else {
				cmdUpgrade = "kubeadm upgrade node --certificate-renewal=false"
				err = d.SSH.CmdAsync(ip, cmdUpgrade)
				if err != nil {
					logger.Error("kubeadm upgrade err: ", err)
					errList = append(errList, err)
				}
			}

			// third to restart kubelet
			logger.Info("[%s] third: to restart kubelet on %s", ip, node)
			err = d.SSH.CmdAsync(ip, "systemctl daemon-reload && systemctl restart kubelet")
			if err != nil {
				errList = append(errList, err)
				logger.Error("systemctl daemon-reload && systemctl restart kubelet err: ", err)
			}

			// fourth to judge nodes is ready
			time.Sleep(time.Second * 10)
			k8sNode, _ := client.GetNodeByName(c, node)
			if client.IsNodeReady(*k8sNode) {
				logger.Info("[%s] fourth:  %s nodes is ready", ip, node)

				// fifth to uncordon node
				err = client.CordonUnCordon(c, node, false)
				if err != nil {
					logger.Error(`k8s.CordonUnCordon err: %s, \n After upgrade,  please run "kubectl uncordon %s" to enable Scheduling`, err, node)
					errList = append(errList, err)
				}
				logger.Info("[%s] fifth: to uncordon node, 10 seconds to wait for %s uncordon", ip, node)
			} else {
				logger.Error("fourth:  %s nodes is not ready, please check the nodes logs to find out reason", node)
			}
		}(host)
	}
	wg.Wait()

	if len(errList) >= 1 {
		return fmt.Errorf("your hava upgrade error. please check upgrade logs")
	}

	return nil
}
