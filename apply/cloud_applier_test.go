package apply

import (
	"testing"

	"gitlab.alibaba-inc.com/seadent/pkg/utils"
)

func TestAppendFile(t *testing.T) {
	type args struct {
		content  string
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"add hosts",
			args{
				content:  "127.0.0.1 localhost",
				fileName: "./test/hosts1",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.AppendFile(tt.args.fileName, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("AppendFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveFileContent(t *testing.T) {
	type args struct {
		fileName string
		content  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"delete hosts",
			args{
				content:  "127.0.0.1 localhost",
				fileName: "./test/hosts1",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.RemoveFileContent(tt.args.fileName, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("RemoveFileContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
