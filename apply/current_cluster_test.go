package apply

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestGetCurrentCluster(t *testing.T) {
	tests := []struct {
		name    string
		want    *v1.Cluster
		wantErr bool
	}{
		{
			"test get cluster nodes",
			&v1.Cluster{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.ClusterSpec{
					Masters: v1.Hosts{
						IPList: []string{},
					},
					Nodes: v1.Hosts{
						IPList: []string{},
					},
				},
				Status: v1.ClusterStatus{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCurrentCluster()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			logger.Info("masters : %v nodes : %v", got.Spec.Masters.IPList, got.Spec.Nodes.IPList)
		})
	}
}
