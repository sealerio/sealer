// Copyright © 2022 Alibaba Group Holding Ltd.
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

package hostpool

import (
	"fmt"
	"net"
	"strconv"

	goscp "github.com/bramvdbogaerde/go-scp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Host contains both static and dynamic information of a host machine.
// Static part: the host config
// dynamic part, including ssh client and sftp client.
type Host struct {
	config HostConfig

	// sshClient is used to create ssh.Session.
	// TODO: remove this and just make ssh.Session remain.
	sshClient *ssh.Client
	// sshSession is created by ssh.Client and used for command execution on specified host.
	sshSession *ssh.Session
	// sftpClient is used to file remote operation on specified host except scp operation.
	sftpClient *sftp.Client
	// scpClient is used to scp files between sealer node and all nodes.
	scpClient *goscp.Client

	// isLocal identifies that whether the initialized host is the sealer binary located node.
	isLocal bool
}

// HostConfig is the host config, including IP, port, login credentials and so on.
type HostConfig struct {
	// IP is the IP address of host.
	// It supports both IPv4 and IPv6.
	IP net.IP

	// Port is the port config used by ssh to connect host
	// The connecting operation will use port 22 if port is not set.
	Port int

	// Usually User will be root. If it is set a non-root user,
	// then this non-root must has a sudo permission.
	User     string
	Password string

	// Encrypted means the password is encrypted.
	// Password above should be decrypted first before being called.
	Encrypted bool

	// TODO: add PkFile support
	// PkFile     string
	// PkPassword string
}

// Initialize setups ssh and sftp clients.
func (host *Host) Initialize() error {
	config := &ssh.ClientConfig{
		User: host.config.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(host.config.Password),
		},
		HostKeyCallback: nil,
	}

	hostAddr := host.config.IP.String()
	port := strconv.Itoa(host.config.Port)

	// sshClient
	sshClient, err := ssh.Dial("tcp", net.JoinHostPort(hostAddr, port), config)
	if err != nil {
		return fmt.Errorf("failed to create ssh client for host(%s): %v", hostAddr, err)
	}
	host.sshClient = sshClient

	// sshSession
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create ssh session for host(%s): %v", hostAddr, err)
	}
	host.sshSession = sshSession

	// sftpClient
	sftpClient, err := sftp.NewClient(sshClient, nil)
	if err != nil {
		return fmt.Errorf("failed to create sftp client for host(%s): %v", hostAddr, err)
	}
	host.sftpClient = sftpClient

	// scpClient
	scpClient, err := goscp.NewClientBySSH(sshClient)
	if err != nil {
		return fmt.Errorf("failed to create scp client for host(%s): %v", hostAddr, err)
	}
	host.scpClient = &scpClient

	// TODO: set isLocal

	return nil
}
