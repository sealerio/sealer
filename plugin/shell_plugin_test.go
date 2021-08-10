package plugin

import (
	"testing"

	typev1 "github.com/alibaba/sealer/types/api/v1"
)

/*

apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PostInstall
  on: role=master
  data: |
     kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule

*/
func TestSheller_Run(t *testing.T) {
	type args struct {
		context Context
		phase   Phase
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"test shell plugin",
			args{
				context: Context{
					Cluster: &typev1.Cluster{
						Spec: typev1.ClusterSpec{
							SSH: typev1.SSH{
								User:     "root",
								Passwd:   "7758521",
								Pk:       "",
								PkPasswd: "",
							},
							Nodes: typev1.Hosts{
								CPU:        "",
								Memory:     "",
								Count:      "",
								SystemDisk: "",
								DataDisks: []string{
									"",
								},
								IPList: []string{
									"192.168.59.11",
								},
							},
						},
					},
					Plugin: &typev1.Plugin{
						Spec: typev1.PluginSpec{
							Data: "ifconfig",
						},
					},
				},
				phase: PhasePostInstall,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Sheller{}
			if err := s.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
