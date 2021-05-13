package utils

import (
	"testing"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestImage(t *testing.T) {
	type args struct {
		cluster *v1.Cluster
	}
	tests := []struct {
		name string
		arg  args
	}{
		{
			name: "test cluster image kuberentes:v1.18.6",
			arg: args{
				cluster: &v1.Cluster{
					Spec: v1.ClusterSpec{
						Image: "kuberentes:v1.18.6",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if image, err := GetImage(tt.arg.cluster.Spec.Image); err != nil {
				t.Errorf("%s failed,%s", tt.name, err)
			} else if image.Name != tt.arg.cluster.Spec.Image {
				t.Errorf("%s failed,cluster Image:%s is not equal to the Image:%s", tt.name, tt.arg.cluster.Spec.Image, image.Name)
			}
		})
	}
}

func TestSetImageMetadata(t *testing.T) {
	type args struct {
		ImageMetadata
	}
	tests := []struct {
		name string
		arg  args
	}{
		{
			name: "test set image kuberentes:v1.18.99",
			arg: args{
				ImageMetadata{
					Name: "kuberentes:v1.18.99",
					ID:   "f6de07561db99",
				},
			},
		},
		{
			name: "test set image kuberentes:v1.18.99",
			arg: args{
				ImageMetadata{
					Name: "kuberentes:v1.18.99",
					ID:   "f6de07561db98",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetImageMetadata(tt.arg.ImageMetadata); err != nil {
				t.Errorf("%s failed :%s", tt.name, err)
			}
		})
	}
}
