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

package ssh

import (
	"net"
	"reflect"
	"testing"

	"github.com/pkg/sftp"
)

/*
func TestSSHCopyLocalToRemote(t *testing.T) {
	type args struct {
		host       string
		localPath  string
		remotePath string
	}
	var (
		host = "10.96.33.168"
		ssh  = SSH{
			User:       "root",
			Password:   "123456",
			PkFile:     "",
			PkPassword: "",
			Timeout:    nil,
		}
	)
	tests := []struct {
		name    string
		fields  SSH
		args    args
		wantErr bool
	}{
		{
			name:   "test for copy file to remote server",
			fields: ssh,
			args: args{
				host,
				"../test/file/01",
				"/home/temp/01",
			},
			wantErr: false,
		},
		{
			name:   "test copy for file when local file is not exist",
			fields: ssh,
			args: args{
				host,
				// local file  001 is not exist.
				"../test/file/123",
				"/home/temp/01",
			},
			wantErr: false,
		},
		{
			name:   "test copy dir to remote server",
			fields: ssh,
			args: args{
				host,
				"../test/file",
				"/home/temp011",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := &SSH{
				User:       tt.fields.User,
				Password:   tt.fields.Password,
				PkFile:     tt.fields.PkFile,
				PkPassword: tt.fields.PkPassword,
				Timeout:    tt.fields.Timeout,
				Fs:         fs.NewFilesystem(),
			}

			if !fileExist(tt.args.localPath) {
				logrus.Error("local filepath is not exit")
				return
			}
			//if ss.IsFileExist(host, tt.args.remotePath) {
			//	log.Error("remote filepath is exit")
			//	return
			//}
			// test copy dir
			err := ss.Copy(tt.args.host, tt.args.localPath, tt.args.remotePath)
			if (err != nil) != tt.wantErr {
				logrus.Error(err)
				t.Errorf("err: %v", err)
			}

			// test the copy result
			//ss.Cmd(tt.args.host, "ls -lh "+tt.args.remotePath)

			// rm remote file
			//ss.Cmd(tt.args.host, "rm -rf "+tt.args.remotePath)
		})
	}
}*/

//func TestSSHFetchRemoteToLocal(t *testing.T) {
//	type args struct {
//		host       net.IP
//		localPath  string
//		remotePath string
//	}
//	var (
//		host = net.IP{}
//		ssh  = SSH{
//			User:       "root",
//			Password:   "",
//			PkFile:     "",
//			PkPassword: "",
//			Timeout:    nil,
//		}
//	)
//	tests := []struct {
//		name    string
//		fields  SSH
//		args    args
//		wantErr bool
//	}{
//		{
//			name:   "test for fetch remote file to local",
//			fields: ssh,
//			args: args{
//				host,
//				"/root/.kube/config",
//				"/root/Clusterfile",
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ss := &SSH{
//				User:       tt.fields.User,
//				Password:   tt.fields.Password,
//				PkFile:     tt.fields.PkFile,
//				PkPassword: tt.fields.PkPassword,
//				Timeout:    tt.fields.Timeout,
//				Fs:         fs.NewFilesystem(),
//			}
//
//			if exist, err := ss.IsFileExist(host, tt.args.remotePath); err != nil {
//				logrus.Error("err: ", err)
//				return
//			} else if !exist {
//				logrus.Error("remote filepath is not exit")
//				return
//			}
//			err := ss.CopyR(tt.args.host, tt.args.localPath, tt.args.remotePath)
//			if (err != nil) != tt.wantErr {
//				logrus.Error(err)
//				t.Errorf("err: %v", err)
//			}
//		})
//	}
//}

/*
func TestSSH_Copy(t *testing.T) {
	type fields struct {
		User       string
		Password   string
		PkFile     string
		PkPassword string
	}
	type args struct {
		host       string
		localPath  string
		remotePath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"test copy dir",
			fields{
				User:       "root",
				Password:   "",
				PkFile:     "",
				PkPassword: "",
			},
			args{
				host:       "",
				localPath:  "./pkg/cert/pki",
				remotePath: "/root/kubernetes/pki",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SSH{
				User:       tt.fields.User,
				Password:   tt.fields.Password,
				PkFile:     tt.fields.PkFile,
				PkPassword: tt.fields.PkPassword,
				Fs:         fs.NewFilesystem(),
			}
			if err := s.Copy(tt.args.host, tt.args.localPath, tt.args.remotePath); (err != nil) != tt.wantErr {
				t.Errorf("Copy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
*/

func Test_epuRWMap_Get(t *testing.T) {
	type args struct {
		k string
	}
	tests := []struct {
		name  string
		m     *epuRWMap
		args  args
		want  *easyProgressUtil
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.m.Get(tt.args.k)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("epuRWMap.Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("epuRWMap.Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_epuRWMap_Set(t *testing.T) {
	type args struct {
		k string
		v *easyProgressUtil
	}
	tests := []struct {
		name string
		m    *epuRWMap
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.Set(tt.args.k, tt.args.v)
		})
	}
}

func Test_easyProgressUtil_increment(t *testing.T) {
	tests := []struct {
		name string
		epu  *easyProgressUtil
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.epu.increment()
		})
	}
}

func Test_easyProgressUtil_fail(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		epu  *easyProgressUtil
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.epu.fail(tt.args.err)
		})
	}
}

func Test_easyProgressUtil_startMessage(t *testing.T) {
	tests := []struct {
		name string
		epu  *easyProgressUtil
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.epu.startMessage()
		})
	}
}

func TestSSH_CopyR(t *testing.T) {
	type args struct {
		host           net.IP
		localFilePath  string
		remoteFilePath string
	}
	tests := []struct {
		name    string
		s       *SSH
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.CopyR(tt.args.host, tt.args.localFilePath, tt.args.remoteFilePath); (err != nil) != tt.wantErr {
				t.Errorf("SSH.CopyR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSH_Copy(t *testing.T) {
	type args struct {
		host       net.IP
		localPath  string
		remotePath string
	}
	tests := []struct {
		name    string
		s       *SSH
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Copy(tt.args.host, tt.args.localPath, tt.args.remotePath); (err != nil) != tt.wantErr {
				t.Errorf("SSH.Copy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSH_remoteMd5Sum(t *testing.T) {
	type args struct {
		host           net.IP
		remoteFilePath string
	}
	tests := []struct {
		name string
		s    *SSH
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.remoteMd5Sum(tt.args.host, tt.args.remoteFilePath); got != tt.want {
				t.Errorf("SSH.remoteMd5Sum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSH_copyLocalDirToRemote(t *testing.T) {
	type args struct {
		host       net.IP
		sftpClient *sftp.Client
		localPath  string
		remotePath string
		epu        *easyProgressUtil
	}
	tests := []struct {
		name string
		s    *SSH
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.copyLocalDirToRemote(tt.args.host, tt.args.sftpClient, tt.args.localPath, tt.args.remotePath, tt.args.epu)
		})
	}
}

func TestSSH_copyLocalFileToRemote(t *testing.T) {
	type args struct {
		host       net.IP
		sftpClient *sftp.Client
		localPath  string
		remotePath string
	}
	tests := []struct {
		name    string
		s       *SSH
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.copyLocalFileToRemote(tt.args.host, tt.args.sftpClient, tt.args.localPath, tt.args.remotePath); (err != nil) != tt.wantErr {
				t.Errorf("SSH.copyLocalFileToRemote() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSH_RemoteDirExist(t *testing.T) {
	type args struct {
		host          net.IP
		remoteDirPath string
	}
	tests := []struct {
		name    string
		s       *SSH
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.RemoteDirExist(tt.args.host, tt.args.remoteDirPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSH.RemoteDirExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SSH.RemoteDirExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSH_IsFileExist(t *testing.T) {
	type args struct {
		host           net.IP
		remoteFilePath string
	}
	tests := []struct {
		name    string
		s       *SSH
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.IsFileExist(tt.args.host, tt.args.remoteFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSH.IsFileExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SSH.IsFileExist() = %v, want %v", got, tt.want)
			}
		})
	}
}
