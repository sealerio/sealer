package guest

import (
	v1 "github.com/alibaba/sealer/types/api/v1"
	"testing"
)

func TestDefault_Apply(t *testing.T) {
	type args struct {
		Cluster *v1.Cluster
	}
	tests := []struct {
		name    string
		args    args
		wanterr bool
	}{
		{
			name: "Master exec cmd : echo 'guest_test success",
			args: args{
				Cluster: &v1.Cluster{
					Spec: v1.ClusterSpec{
						Image: "kuberentes:v1.18.6",
						SSH: v1.SSH{
							User:     "root",
							Passwd:   "huaijiahui.com",
							Pk:       "",
							PkPasswd: "",
						},
						Masters: v1.Hosts{
							IPList: []string{"192.168.56.104"},
						},
					},
				},
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Default := NewGuestManager()
			if err := Default.Apply(tt.args.Cluster); (err != nil) != tt.wanterr {
				t.Errorf("Apply failed, %s", err)
			}
		})
	}
}
