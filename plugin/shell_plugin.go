package plugin

import (
	"github.com/alibaba/sealer/utils"
)

type Sheller struct {
	role string
	data string
}
func (s Sheller) Run(context Context, phase Phase){
	s.SSH.CmdAsync()
}