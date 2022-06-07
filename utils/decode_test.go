// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

/*func TestDecodeCRDFromFile(t *testing.T) {
	type args struct {
		filepath string
		kind     string
	}
	testFile := "test/file/Clusterfile"
	var tests = []struct {
		name string
		args args
	}{
		{
			"test " + common.InitConfiguration,
			args{
				filepath: testFile,
				kind:     common.InitConfiguration,
			},
		},
		{
			"test " + common.JoinConfiguration,
			args{
				filepath: testFile,
				kind:     common.JoinConfiguration,
			},
		},
		{
			"test " + common.KubeletConfiguration,
			args{
				filepath: testFile,
				kind:     common.KubeletConfiguration,
			},
		},
		{
			"test " + common.Cluster,
			args{
				filepath: testFile,
				kind:     common.Cluster,
			},
		},
		{
			"test " + common.Plugin,
			args{
				filepath: testFile,
				kind:     common.Plugin,
			},
		},
		{
			"test " + common.Config,
			args{
				filepath: testFile,
				kind:     common.Config,
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

func TestDecodeV1ClusterFromFile(t *testing.T) {
	testFile := "test/file/v1Clusterfile"
	t.Run("test decode v1 cluster"+testFile, func(t *testing.T) {
		got, err := DecodeV1ClusterFromFile(testFile)
		if err != nil {
			t.Errorf("failed to decode v1 cluster form %s: %v", testFile, err)
		}
		fmt.Printf("got:\n %#+v", got)
	})
}*/
