package yaml

import (
	"fmt"
	"testing"
)

func TestDecodeCRDFromFile(t *testing.T) {
	type args struct {
		filepath string
		kind     string
	}

	var tests = []struct {
		name string
		args args
	}{
		{
			"test",
			args{
				filepath: "../test/file/Clusterfile",
				kind:     "InitConfiguration",
			},
		},
		{
			"test",
			args{
				filepath: "../test/file/Clusterfile",
				kind:     "Cluster",
			},
		},
		{
			"test",
			args{
				filepath: "../test/file/Clusterfile",
				kind:     "Plugin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if obj, err := DecodeCRDFromFile(tt.args.filepath, tt.args.kind); err != nil {
				t.Errorf("DecodeV1CRD1() error = %v", err)
			} else {
				fmt.Printf("%#+v \n", obj)
			}
		})
	}
}
