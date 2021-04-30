package runtime

import "testing"

func TestVerionCompare(t *testing.T) {
	type args struct {
		v1 string
		v2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"test version",
			args{
				v1: "v1.20.0",
				v2: "v1.19.1",
			},
			true,
		},
		{
			"test version",
			args{
				v1: "v1.20.0",
				v2: "v1.20.0",
			},
			true,
		},
		{
			"test version",
			args{
				v1: "v2.10.0",
				v2: "v1.20.0",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VersionCompare(tt.args.v1, tt.args.v2); got != tt.want {
				t.Errorf("VerionCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}
