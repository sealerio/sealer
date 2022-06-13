// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/6/13 2:41 PM
// @File : iputils_test.go
//

package net

import "testing"

func TestAssemblyIPList(t *testing.T) {
	type args struct {
		args *string
	}
	ta := "172.16.0.11"
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args:    args{args: &ta},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AssemblyIPList(tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("AssemblyIPList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
