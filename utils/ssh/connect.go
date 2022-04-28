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
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"

	"github.com/alibaba/sealer/utils"
	"golang.org/x/crypto/ssh"
)

const DefaultSSHPort = "22"

func (s *SSH) connect(host string) (*ssh.Client, error) {
	if s.Encrypted {
		passwd, err := utils.AesDecrypt([]byte(s.Password))
		if err != nil {
			return nil, err
		}
		s.Password = passwd
		s.Encrypted = false
	}
	auth := s.sshAuthMethod(s.Password, s.PkFile, s.PkPassword)
	config := ssh.Config{
		Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
	}
	DefaultTimeout := time.Duration(15) * time.Second
	if s.Timeout == nil {
		s.Timeout = &DefaultTimeout
	}
	clientConfig := &ssh.ClientConfig{
		User:    s.User,
		Auth:    auth,
		Timeout: *s.Timeout,
		Config:  config,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	if s.Port == "" {
		s.Port = DefaultSSHPort
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:%s", host, s.Port), clientConfig)
}

func (s *SSH) Connect(host string) (*ssh.Client, *ssh.Session, error) {
	client, err := s.connect(host)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     //disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, err
	}

	return client, session, nil
}

func (s *SSH) sshAuthMethod(password, pkFile, pkPasswd string) (auth []ssh.AuthMethod) {
	if fileExist(pkFile) {
		am, err := s.sshPrivateKeyMethod(pkFile, pkPasswd)
		if err == nil {
			auth = append(auth, am)
		}
	}
	if password != "" {
		auth = append(auth, s.sshPasswordMethod(password))
	}
	return auth
}

//Authentication with a private key,private key has password and no password to verify in this
func (s *SSH) sshPrivateKeyMethod(pkFile, pkPassword string) (am ssh.AuthMethod, err error) {
	pkData, err := ioutil.ReadFile(filepath.Clean(pkFile))
	if err != nil {
		return nil, err
	}

	var pk ssh.Signer
	if pkPassword == "" {
		pk, err = ssh.ParsePrivateKey(pkData)
		if err != nil {
			return nil, err
		}
	} else {
		bufPwd := []byte(pkPassword)
		pk, err = ssh.ParsePrivateKeyWithPassphrase(pkData, bufPwd)
		if err != nil {
			return nil, err
		}
	}
	return ssh.PublicKeys(pk), nil
}

func (s *SSH) sshPasswordMethod(password string) ssh.AuthMethod {
	return ssh.Password(password)
}

func (s *SSH) sftpConnect(host string) (*ssh.Client, *sftp.Client, error) {
	sshClient, err := s.connect(host)
	if err != nil {
		return nil, nil, err
	}

	// create sftp client
	sftpClient, err := sftp.NewClient(sshClient)
	return sshClient, sftpClient, err
}
