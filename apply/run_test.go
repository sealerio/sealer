package apply

import (
	"github.com/alibaba/sealer/common"

	"testing"

	"github.com/alibaba/sealer/logger"
)

func TestAssemblyIPList(t *testing.T) {
	tests := []struct {
		name    string
		args    *common.RunArgs
		wantErr bool
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
			false,
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
			true,
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
			true,
		},
		{
			"errorData3",
			&common.RunArgs{
				Masters:    "-10.110.101.",
				Nodes:      "10.110.101.1-",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
			true,
		},
		{
			"errorData4",
			&common.RunArgs{
				Masters:    "a-b",
				Nodes:      "a-",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AssemblyIPList(&tt.args.Masters); (err != nil) != tt.wantErr {
				logger.Error("masters : %v , nodes : %v", &tt.args.Masters, &tt.args.Nodes)
			}
			logger.Info("masters : %v , nodes : %v", &tt.args.Masters, &tt.args.Nodes)
		})
	}
}
