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
			"baseData",
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
		{
			"errorData",
			&common.RunArgs{
				Masters:    "10.110.101.10-10.110.101.5",
				Nodes:      "10.110.101.1-10.110.101.5",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
		},
		{
			"errorData2",
			&common.RunArgs{
				Masters:    "10.110.101.10-10.110.101.5-10.110.101.55",
				Nodes:      "10.110.101.1-10.110.101.5",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
		},
		{
			"errorData3",
			&common.RunArgs{
				Masters:    "-10.110.101.5",
				Nodes:      "10.110.101.1-",
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
			logger.Info("masters : %v , nodes : %v", &tt.args.Masters, &tt.args.Nodes)
		})
	}
}
