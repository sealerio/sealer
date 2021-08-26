package apply

import (
	"github.com/alibaba/sealer/common"

	"testing"

	"github.com/alibaba/sealer/logger"
)

func TestAssemblyIPList(t *testing.T) {
	tests := []struct {
		name string
		args *common.RunArgs
	}{
		{
			"TestAssemblyIPList",
			&common.RunArgs{
				Masters:    "10.110.101.1-10.110.101.5",
				Nodes:      "10.110.101.1-10.110.101.5",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssemblyIPList(&tt.args.Masters)
			logger.Info("masters : %v ", &tt.args.Masters)
		})
	}
}
