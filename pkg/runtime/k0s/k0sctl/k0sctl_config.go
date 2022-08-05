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

package k0sctl

import (
	"fmt"
	"strconv"

	"github.com/sealerio/sealer/pkg/runtime/k0s/k0sctl/v1beta1"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utilsnet "github.com/sealerio/sealer/utils/net"
	osi "github.com/sealerio/sealer/utils/os"

	"github.com/k0sproject/dig"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
	"github.com/k0sproject/rig"
	yaml2 "gopkg.in/yaml.v2"
)

type K0sConfig struct {
	*v1beta1.Cluster
}

// ConvertTok0sConfig convert cluster file spec host to k0sctl spec hosts.
func (c *K0sConfig) ConvertTok0sConfig(clusterFile *v2.Cluster) error {
	return c.convertIPVSToAddress(clusterFile)
}

func (c *K0sConfig) convertIPVSToAddress(clusterFile *v2.Cluster) error {
	masterIPList := utilsnet.IPsToIPStrs(clusterFile.GetIPSByRole(ClusterFileRoleMaster))
	if err := c.joinK0sHosts(masterIPList, clusterFile.Spec.Hosts, clusterFile.Spec.SSH, K0sController); err != nil {
		return err
	}

	nodeIPList := utilsnet.IPsToIPStrs(clusterFile.GetIPSByRole(ClusterFileRoleWorker))
	if err := c.joinK0sHosts(nodeIPList, clusterFile.Spec.Hosts, clusterFile.Spec.SSH, K0sWorker); err != nil {
		return err
	}
	// TODO: Get the controller+worker role node.
	// because sealer's cluster file do not support master and worker roles,
	// so we can't generate a controller and worker role in k0s config.
	return nil
}

func (c *K0sConfig) joinK0sHosts(ipList []string, hosts []v2.Host, ssh v1.SSH, role string) error {
	for _, ip := range ipList {
		port, err := c.parseSSHPortByIP(ip, hosts, ssh)
		if err != nil {
			return err
		}
		//Now sealer do not support rootless, so default user is always 'root'.
		c.addHostField(ip, port, role, K0sDefaultUser)
	}
	return nil
}

// addHostField generate k0s config spec Host and append it to config file.
func (c *K0sConfig) addHostField(ipAddr string, port int, role, user string) {
	host := cluster.Host{
		Connection: rig.Connection{
			SSH: &rig.SSH{
				Address: ipAddr,
				Port:    port,
				User:    user,
			},
		},
		Role:          role,
		UploadBinary:  true,
		K0sBinaryPath: K0sUploadBinaryPath,
	}
	c.Spec.Hosts = append(c.Spec.Hosts, &host)
}

// parseSSHPortByIP parse cluster file ssh port to k0s ssh port.
func (c *K0sConfig) parseSSHPortByIP(ipAddr string, hosts []v2.Host, ssh v1.SSH) (int, error) {
	hostsMap := c.getHostsMap(hosts)
	host := hostsMap[ipAddr]
	if host.SSH.Port != "" {
		return strconv.Atoi(host.SSH.Port)
	}
	return strconv.Atoi(ssh.Port)
}

// getHostsMap convert hosts to map[string]v1.SSH with ssh arg
func (c *K0sConfig) getHostsMap(hosts []v2.Host) map[string]v2.Host {
	hostsMap := make(map[string]v2.Host)
	for _, host := range hosts {
		ips := utilsnet.IPsToIPStrs(host.IPS)
		for _, ip := range ips {
			hostsMap[ip] = host
		}
	}
	return hostsMap
}

// DefineConfigFork0s define the private registry for image repo such as: sea.hub:5000
func (c *K0sConfig) DefineConfigFork0s(version, domain, port, name string) {
	c.Spec.K0s = &cluster.K0s{
		Version: version,
		Config: dig.Mapping{
			"apiVersion": "k0s.k0sproject.io/v1beta1",
			"metadata": dig.Mapping{
				"name": name,
			},
			"spec": dig.Mapping{
				"images": dig.Mapping{
					"repository": "\"" + domain + ":" + port + "\"",
				},
			},
		},
	}
}

// WriteConfigToMaster0 write k0sctl config to rootfs
func (c *K0sConfig) WriteConfigToMaster0(rootfs string) error {
	k0sConfig := c.Cluster
	marshal, err := yaml2.Marshal(k0sConfig)
	if err != nil {
		return err
	}
	if err := osi.NewAtomicWriter(rootfs + "/k0sctl.yaml").WriteFile(marshal); err != nil {
		return fmt.Errorf("failed to write k0sctl config: %v ", err)
	}
	return nil
}
