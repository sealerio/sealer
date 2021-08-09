package plugin

import (
	"fmt"

	"github.com/alibaba/sealer/utils/ssh"
)

type Sheller struct {
}

func (s Sheller) Run(context Context, phase Phase) error {
	//get cmdline content
	pluginData := context.Plugin.Spec.Data
	//get all host ip
	masterIP := context.Cluster.Spec.Masters.IPList
	nodeIP := context.Cluster.Spec.Nodes.IPList
	hostIP := append(masterIP, nodeIP...)

	SSH := ssh.NewSSHByCluster(context.Cluster)

	for i := 0; i < len(hostIP); i++ {
		err := SSH.CmdAsync(hostIP[i], pluginData)
		if err != nil {
			return fmt.Errorf("failed to xxx %v", err)
		}
	}
}
