package plugin

import (
	"github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type Sheller struct {
}

func (s Sheller) Run(context Context, phase Phase) {
	var ps v1.PluginSpec
	cmds := ps.Data

	var hs v1.Hosts
	host := hs.IPList

	ssh := NewSSHByCluster(context.Cluster)
	for i:= 0, i< len(host), i++ {
		ssh.CmdAsync(host[i], cmds)
	}
}