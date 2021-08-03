package plugin

import (
	"fmt"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"
)

type Sheller struct {
}

func (s Sheller) Run(context Context, phase Phase) {
	var ps v1.PluginSpec
	cmds := ps.Data

	var hs v1.Hosts
	host := hs.IPList

	ssh := ssh.NewSSHByCluster(context.Cluster)

	for i := 0; i < len(host); i++ {
		err := ssh.CmdAsync(host[i], cmds)
		if err == nil {
			fmt.Printf("err is nil\\n")
		} else {
			fmt.Printf("err is %v\\n", err)
		}
	}
}
