package ssh

import (
	"testing"
)

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
				ssh: SSH{
					"root",
					"huaijiahui.com",
					"",
					"",
					nil,
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
				ssh: SSH{
					"root",
					"huaijiahui.com",
					"",
					"",
					nil,
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
				ssh: SSH{
					"root",
					"huaijiahui.com",
					"",
					"",
					nil,
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
				ssh: SSH{
					"root",
					"huaijiahui.com",
					"",
					"",
					nil,
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
				ssh: SSH{
					"root",
					"huaijiahui.com",
					"",
					"",
					nil,
				},
				host: "192.168.56.103",
				cmd:  "bash /opt/touchTxt.sh",
			},
			wantErr: false,
		},
		{
			name: "ls /opt/test",
			args: args{
				ssh: SSH{
					"root",
					"huaijiahui.com",
					"",
					"",
					nil,
				},
				host: "192.168.56.103",
				cmd:  "ls /opt/test",
			},
			wantErr: false,
		},
		{
			name: "remove test.txt",
			args: args{
				ssh: SSH{
					"root",
					"huaijiahui.com",
					"",
					"",
					nil,
				},
				host: "192.168.56.103",
				cmd:  "bash /opt/removeTxt.sh",
			},
			wantErr: false,
		},
		{
			name: "exist 1",
			args: args{
				ssh: SSH{
					"root",
					"huaijiahui.com",
					"",
					"",
					nil,
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
