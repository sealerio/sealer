package plugin

import (
	"fmt"

	"github.com/alibaba/sealer/utils/ssh"
)

type Sheller struct {
}

func (s Sheller) Run(context Context, phase Phase) error {
	if string(phase) != context.Plugin.Spec.On {
		return nil
	}
	//get cmdline content
	pluginData := context.Plugin.Spec.Data
	//get all host ip
	masterIP := context.Cluster.Spec.Masters.IPList
	nodeIP := context.Cluster.Spec.Nodes.IPList
	hostIP := append(masterIP, nodeIP...)

	SSH := ssh.NewSSHByCluster(context.Cluster)
	for _, ip := range hostIP {
		err := SSH.CmdAsync(ip, pluginData)
		if err != nil {
			return fmt.Errorf("failed to run shell cmd,  %v", err)
		}
	}
	return nil
}
