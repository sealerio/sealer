package command

import (
	"fmt"

	"github.com/alibaba/sealer/common"
)

/*
  exec some commands on remote host like create ipvs rules or add routers.
  create certs on master1 master2.
*/
func RemoteCerts(altNames []string, hostIP, hostName, serviceCIRD, DNSDomain string) string {
	cmd := "seautil certs "
	if hostIP != "" {
		cmd += fmt.Sprintf(" --node-ip %s", hostIP)
	}

	if hostName != "" {
		cmd += fmt.Sprintf(" --node-name %s", hostName)
	}

	if serviceCIRD != "" {
		cmd += fmt.Sprintf(" --service-cidr %s", serviceCIRD)
	}

	if DNSDomain != "" {
		cmd += fmt.Sprintf(" --dns-domain %s", DNSDomain)
	}

	for _, name := range append(altNames, common.APIServerDomain) {
		if name != "" {
			cmd += fmt.Sprintf(" --alt-names %s", name)
		}
	}

	return cmd
}
