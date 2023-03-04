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
)

/*
func TestSSH_Cmd(t *testing.T) {
	type args struct {
		ssh       SSH
		host, cmd string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "touch test.txt",
			args: args{
				ssh: SSH{false,
					false,
					"root",
					"huaijiahui.com",
					"",
					"",
					"",
					nil,
					[]net.Addr{}, nil,
				},
				host: "192.168.56.103",
				cmd:  "bash /opt/touchTxt.sh",
			},
			want:    "success touch test.txt\r\n", //命令返回值后缀为/r/n
			wantErr: false,
		},
		{
			name: "ls /opt/test",
			args: args{
				ssh: SSH{false,
					false,
					"root",
					"huaijiahui.com",
					"",
					"",
					"",
					nil,
					[]net.Addr{}, nil,
				},
				host: "192.168.56.103",
				cmd:  "ls /opt/test",
			},
			want:    "test.txt\r\n",
			wantErr: false,
		},
		{
			name: "remove test.txt",
			args: args{
				ssh: SSH{false,
					false,
					"root",
					"huaijiahui.com",
					"",
					"",
					"",
					nil,
					[]net.Addr{}, nil,
				},
				host: "192.168.56.103",
				cmd:  "bash /opt/removeTxt.sh",
			},
			want:    "test remove success\r\n",
			wantErr: false,
		},
		{
			name: "exist 1",
			args: args{
				ssh: SSH{false,
					false,
					"root",
					"huaijiahui.com",
					"",
					"",
					"",
					nil,
					[]net.Addr{}, nil,
				},
				host: "192.168.56.103",
				cmd:  "bash /opt/exit1.sh",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.ssh.Cmd(tt.args.host, tt.args.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cmd err : %v,  wangErr is %v", err, tt.wantErr)
			}

			if string(got) != tt.want {
				t.Errorf("got={%s},want={%s}", string(got), tt.want)
			}
		})
	}
}

func TestSSH_CmdAsync(t *testing.T) {
	type args struct {
		ssh       SSH
		host, cmd string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "touch test.txt",
			args: args{
				ssh: SSH{false,
					false,
					"root",
					"huaijiahui.com",
					"",
					"",
					"",
					nil,
					[]net.Addr{}, nil,
				},
				host: "192.168.56.103",
				cmd:  "bash /opt/touchTxt.sh",
			},
			wantErr: false,
		},
		{
			name: "ls /opt/test",
			args: args{
				ssh: SSH{false,
					false,
					"root",
					"huaijiahui.com",
					"",
					"",
					"",
					nil,
					[]net.Addr{}, nil,
				},
				host: "192.168.56.103",
				cmd:  "ls /opt/test",
			},
			wantErr: false,
		},
		{
			name: "remove test.txt",
			args: args{
				ssh: SSH{false,
					false,
					"root",
					"huaijiahui.com",
					"",
					"",
					"",
					nil,
					[]net.Addr{}, nil,
				},
				host: "192.168.56.103",
				cmd:  "bash /opt/removeTxt.sh",
			},
			wantErr: false,
		},
		{
			name: "exist 1",
			args: args{
				ssh: SSH{false,
					false,
					"root",
					"huaijiahui.com",
					"",
					"",
					"",
					nil,
					[]net.Addr{}, nil,
				},
				host: "192.168.56.103",
				cmd:  "bash /opt/exit1.sh",
			},
			wantErr: true, //Process exited with status 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.args.ssh.CmdAsync(tt.args.host, tt.args.cmd); (err != nil) != tt.wantErr {
				t.Errorf("Cmd err : %v,  wangErr is %v", err, tt.wantErr)
			}
		})
	}
}
*/

func TestSSH_Ping(t *testing.T) {
	type args struct {
		host net.IP
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
			if err := tt.s.Ping(tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("SSH.Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSH_CmdAsync(t *testing.T) {
	type args struct {
		host net.IP
		cmds []string
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
			if err := tt.s.CmdAsync(tt.args.host, tt.args.cmds...); (err != nil) != tt.wantErr {
				t.Errorf("SSH.CmdAsync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSH_Cmd(t *testing.T) {
	type args struct {
		host net.IP
		cmd  string
	}
	tests := []struct {
		name    string
		s       *SSH
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.Cmd(tt.args.host, tt.args.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSH.Cmd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SSH.Cmd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSH_CmdToString(t *testing.T) {
	type args struct {
		host  net.IP
		cmd   string
		split string
	}
	tests := []struct {
		name    string
		s       *SSH
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.CmdToString(tt.args.host, tt.args.cmd, tt.args.split)
			if (err != nil) != tt.wantErr {
				t.Errorf("SSH.CmdToString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SSH.CmdToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
