package ssh

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/alibaba/sealer/logger"
)

/**
  SSH connection operation
*/
func (s *SSH) connect(host string) (*ssh.Client, error) {
	auth := s.sshAuthMethod(s.Password, s.PkFile, s.PkPassword)
	config := ssh.Config{
		Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
	}
	DefaultTimeout := time.Duration(1) * time.Minute
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
	addr := s.addrReformat(host)
	return ssh.Dial("tcp", addr, clientConfig)
}

func (s *SSH) Connect(host string) (*ssh.Session, error) {
	client, err := s.connect(host)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     //disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return nil, err
	}

	return session, nil
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
	pkData := s.readFile(pkFile)
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

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
func (s *SSH) sshPasswordMethod(password string) ssh.AuthMethod {
	return ssh.Password(password)
}

func (s *SSH) readFile(name string) []byte {
	content, err := ioutil.ReadFile(name)
	if err != nil {
		logger.Error("read [%s] file failed, %s", name, err)
		os.Exit(1)
	}
	return content
}

func (s *SSH) addrReformat(host string) string {
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:22", host)
	}
	return host
}

//RemoteFileExist is
func (s *SSH) IsFileExist(host, remoteFilePath string) bool {
	// if remote file is
	// ls -l | grep aa | wc -l
	remoteFileName := path.Base(remoteFilePath) // aa
	remoteFileDirName := path.Dir(remoteFilePath)
	//it's bug: if file is aa.bak, `ls -l | grep aa | wc -l` is 1 ,should use `ll aa 2>/dev/null |wc -l`
	//remoteFileCommand := fmt.Sprintf("ls -l %s| grep %s | grep -v grep |wc -l", remoteFileDirName, remoteFileName)
	remoteFileCommand := fmt.Sprintf("ls -l %s/%s 2>/dev/null |wc -l", remoteFileDirName, remoteFileName)

	data, err := s.CmdToString(host, remoteFileCommand, " ")
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[ssh][%s]remoteFileCommand err:%s", host, err)
		}
	}()
	if err != nil {
		panic(1)
	}
	count, err := strconv.Atoi(strings.TrimSpace(data))
	defer func() {
		if r := recover(); r != nil {
			logger.Error("[ssh][%s]RemoteFileExist:%s", host, err)
		}
	}()
	if err != nil {
		panic(1)
	}
	return count != 0
}
