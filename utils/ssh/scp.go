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
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/alibaba/sealer/common"

	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"

	dockerstreams "github.com/docker/cli/cli/streams"
	dockerioutils "github.com/docker/docker/pkg/ioutils"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/utils"
)

const KByte = 1024
const MByte = 1024 * 1024

const (
	Md5sumCmd = "md5sum %s | cut -d\" \" -f1"
)

type easyProgressUtil struct {
	output         progress.Output
	copyID         string
	completeNumber int
	total          int
}

func (epu *easyProgressUtil) increment() {
	epu.completeNumber = epu.completeNumber + 1
	progress.Update(epu.output, epu.copyID, fmt.Sprintf("%d/%d", epu.completeNumber, epu.total))
}

func (epu *easyProgressUtil) fail(err error) {
	progress.Update(epu.output, epu.copyID, fmt.Sprintf("failed, err: %s", err))
}

func (epu *easyProgressUtil) startMessage() {
	progress.Update(epu.output, epu.copyID, fmt.Sprintf("%d/%d", epu.completeNumber, epu.total))
}

func (s *SSH) RemoteMd5Sum(host, remoteFilePath string) string {
	cmd := fmt.Sprintf(Md5sumCmd, remoteFilePath)
	remoteMD5, err := s.CmdToString(host, cmd, "")
	if err != nil {
		logger.Error("count remote md5 failed %s %s %v", host, remoteFilePath, err)
	}
	return remoteMD5
}

//CmdToString is in host exec cmd and replace to spilt str
func (s *SSH) CmdToString(host, cmd, spilt string) (string, error) {
	data, err := s.Cmd(host, cmd)
	str := string(data)
	if err != nil {
		return str, fmt.Errorf("exec remote command failed %s %s %s", host, cmd, err)
	}
	if data != nil {
		str = strings.ReplaceAll(str, "\r\n", spilt)
		str = strings.ReplaceAll(str, "\n", spilt)
		return str, nil
	}
	return str, fmt.Errorf("command %s %s return nil", host, cmd)
}

//SftpConnect  is
func (s *SSH) sftpConnect(host string) (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)
	// get auth method
	auth = s.sshAuthMethod(s.Password, s.PkFile, s.PkPassword)

	clientConfig = &ssh.ClientConfig{
		User:    s.User,
		Auth:    auth,
		Timeout: 30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Config: ssh.Config{
			Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
		},
	}

	// connet to ssh
	ip, port := utils.GetSSHHostIPAndPort(host)
	addr = s.addrReformat(ip, port)

	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create sftp client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return sftpClient, nil
}

// CopyRemoteFileToLocal is scp remote file to local
func (s *SSH) Fetch(host, localFilePath, remoteFilePath string) error {
	if utils.IsLocalIP(host, s.LocalAddress) {
		if remoteFilePath != localFilePath {
			logger.Debug("local copy files src %s to dst %s", remoteFilePath, localFilePath)
			return utils.RecursionCopy(remoteFilePath, localFilePath)
		}
		return nil
	}
	sftpClient, err := s.sftpConnect(host)
	if err != nil {
		return fmt.Errorf("new sftp client failed %v", err)
	}
	defer sftpClient.Close()
	// open remote source file
	srcFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		return fmt.Errorf("open remote file failed %v, remote path: %s", err, remoteFilePath)
	}
	defer srcFile.Close()

	err = utils.MkFileFullPathDir(localFilePath)
	if err != nil {
		return err
	}
	// open local Destination file
	dstFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("create local file failed %v", err)
	}
	defer dstFile.Close()
	// copy to local file
	_, err = srcFile.WriteTo(dstFile)
	return err
}

// CopyLocalToRemote is copy file or dir to remotePath, add md5 validate
func (s *SSH) Copy(host, localPath, remotePath string) error {
	if utils.IsLocalIP(host, s.LocalAddress) {
		logger.Debug("local copy files src %s to dst %s", localPath, remotePath)
		return utils.RecursionCopy(localPath, remotePath)
	}
	logger.Debug("remote copy files src %s to dst %s", localPath, remotePath)
	sftpClient, err := s.sftpConnect(host)
	if err != nil {
		return fmt.Errorf("new sftp client failed %s", err)
	}
	sshClient, err := s.connect(host)
	if err != nil {
		return fmt.Errorf("new ssh client failed %s", err)
	}
	defer sftpClient.Close()
	defer sshClient.Close()

	f, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("get file stat failed %s", err)
	}

	baseRemoteFilePath := filepath.Dir(remotePath)
	_, err = sftpClient.ReadDir(baseRemoteFilePath)
	if err != nil {
		if err = sftpClient.MkdirAll(baseRemoteFilePath); err != nil {
			return err
		}
	}
	number := 1
	if f.IsDir() {
		number = utils.CountDirFiles(localPath)
	}
	// no file in dir, do need to send
	if number == 0 {
		return nil
	}
	var (
		reader, writer  = io.Pipe()
		writeFlusher    = dockerioutils.NewWriteFlusher(writer)
		progressChanOut = streamformatter.NewJSONProgressOutput(writeFlusher, false)
		streamOut       = dockerstreams.NewOut(common.StdOut)
		epu             = &easyProgressUtil{
			output:         progressChanOut,
			completeNumber: 0,
			total:          number,
			copyID:         "copying files to " + host,
		}
	)
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
		_ = writeFlusher.Close()
	}()
	go func() {
		err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, streamOut, nil)
		if err != nil && err != io.ErrClosedPipe {
			logger.Warn("error occurs in display progressing, err: %s", err)
		}
	}()

	epu.startMessage()
	if f.IsDir() {
		s.copyLocalDirToRemote(host, sftpClient, localPath, remotePath, epu)
	} else {
		err = s.copyLocalFileToRemote(host, sftpClient, localPath, remotePath)
		if err != nil {
			epu.fail(err)
		}
		epu.increment()
	}
	return nil
}

func (s *SSH) copyLocalDirToRemote(host string, sftpClient *sftp.Client, localPath, remotePath string, epu *easyProgressUtil) {
	localFiles, err := ioutil.ReadDir(localPath)
	if err != nil {
		logger.Error("read local path dir failed %s %s", host, localPath)
		return
	}
	if err = sftpClient.MkdirAll(remotePath); err != nil {
		logger.Error("failed to create remote path %s:%v", remotePath, err)
		return
	}
	for _, file := range localFiles {
		lfp := path.Join(localPath, file.Name())
		rfp := path.Join(remotePath, file.Name())
		if file.IsDir() {
			if err = sftpClient.MkdirAll(rfp); err != nil {
				logger.Error("failed to create remote path %s:%v", rfp, err)
				return
			}
			s.copyLocalDirToRemote(host, sftpClient, lfp, rfp, epu)
		} else {
			err := s.copyLocalFileToRemote(host, sftpClient, lfp, rfp)
			if err != nil {
				errMsg := fmt.Sprintf("copy local file to remote failed %v %s %s %s", err, host, lfp, rfp)
				epu.fail(err)
				logger.Error(errMsg)
				return
			}
			epu.increment()
		}
	}
}

// check the remote file existence before copying
// solve the sesion
func (s *SSH) copyLocalFileToRemote(host string, sftpClient *sftp.Client, localPath, remotePath string) error {
	var (
		srcMd5, dstMd5 string
	)
	srcMd5 = LocalMd5Sum(localPath)
	if s.IsFileExist(host, remotePath) {
		dstMd5 = s.RemoteMd5Sum(host, remotePath)
		if srcMd5 == dstMd5 {
			logger.Debug("remote dst %s already exists and is the latest version , skip copying process", remotePath)
			return nil
		}
	}
	srcFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return err
	}
	fileStat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("get file stat failed %v", err)
	}
	// TODO seems not work
	if err := dstFile.Chmod(fileStat.Mode()); err != nil {
		return fmt.Errorf("chmod remote file failed %v", err)
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	dstMd5 = s.RemoteMd5Sum(host, remotePath)
	if srcMd5 != dstMd5 {
		return fmt.Errorf("[ssh][%s] validate md5sum failed %s != %s", host, srcMd5, dstMd5)
	}
	return nil
}

func LocalMd5Sum(localPath string) string {
	md5, err := utils.FileMD5(localPath)
	if err != nil {
		logger.Error("get file md5 failed %v", err)
		return ""
	}
	return md5
}

//if remote file not exist return false and nil
func (s *SSH) RemoteDirExist(host, remoteDirpath string) (bool, error) {
	sftpClient, err := s.sftpConnect(host)
	if err != nil {
		return false, err
	}
	defer sftpClient.Close()
	if _, err := sftpClient.ReadDir(remoteDirpath); err != nil {
		return false, err
	}
	return true, nil
}
