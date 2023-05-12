// Copyright © 2022 Alibaba Group Holding Ltd.
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

package clusterruntime

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/infradriver"

	"github.com/sirupsen/logrus"
)

func getWorkerIPList(infraDriver infradriver.InfraDriver) []net.IP {
	masters := make(map[string]bool)
	for _, master := range infraDriver.GetHostIPListByRole(common.MASTER) {
		masters[master.String()] = true
	}
	all := infraDriver.GetHostIPList()
	workers := make([]net.IP, len(all)-len(masters))

	index := 0
	for _, ip := range all {
		if !masters[ip.String()] {
			workers[index] = ip
			index++
		}
	}

	return workers
}

// LoadToRegistry just load container image to local registry
func LoadToRegistry(infraDriver infradriver.InfraDriver, distributor imagedistributor.Distributor) error {
	regConfig := infraDriver.GetClusterRegistry()
	// todo only support load image to local registry at present
	if regConfig.LocalRegistry == nil {
		return nil
	}

	deployHosts := infraDriver.GetHostIPListByRole(common.MASTER)
	if len(deployHosts) < 1 {
		return fmt.Errorf("local registry host can not be nil")
	}
	master0 := deployHosts[0]

	logrus.Infof("start to apply with mode(%s)", common.ApplyModeLoadImage)
	if !*regConfig.LocalRegistry.HA {
		deployHosts = []net.IP{master0}
	}

	if err := distributor.DistributeRegistry(deployHosts, filepath.Join(infraDriver.GetClusterRootfsPath(), "registry")); err != nil {
		return err
	}

	logrus.Infof("load image success")
	return nil
}

func CheckNodeSSH(infraDriver infradriver.InfraDriver, clientHosts []net.IP) ([]net.IP, error) {
	var failed []net.IP
	for i := range clientHosts {
		n := clientHosts[i]
		logrus.Debug("checking ssh client of ", n)
		err := infraDriver.CmdAsync(n, nil, "ls >> /dev/null")
		if err != nil {
			failed = append(failed, n)
			logrus.Errorf("failed to connect node %s: %v", n.String(), err)
		}
	}

	var retErr error
	if len(failed) > 0 {
		retErr = fmt.Errorf("failed to connect node: %v, maybe you have change its sshpasswd, if so, please correct passwd via cmd (kubectl -n kube-system edit cm sealer-clusterfile) or check other errors by yourself", failed)
	}
	return failed, retErr
}
