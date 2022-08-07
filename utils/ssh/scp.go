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
	"sync"

	dockerioutils "github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/progress"
	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"

	utilsnet "github.com/sealerio/sealer/utils/net"
	osi "github.com/sealerio/sealer/utils/os"
)

const (
	Md5sumCmd = "md5sum %s | cut -d\" \" -f1"
)

var (
	displayInitOnce sync.Once
	reader          *io.PipeReader
	writer          *io.PipeWriter
	writeFlusher    *dockerioutils.WriteFlusher
	progressChanOut progress.Output
	epuMap          = map[string]*easyProgressUtil{}
)

type easyProgressUtil struct {
	output         progress.Output
	copyID         string
	completeNumber int
	total          int
}

//must call DisplayInit first
func registerEpu(ip net.IP, total int) {
	if progressChanOut == nil {
		logrus.Warn("call DisplayInit first")
		return
	}
	if _, ok := epuMap[ip.String()]; !ok {
		epuMap[ip.String()] = &easyProgressUtil{
			output:         progressChanOut,
			copyID:         "copying files to " + ip.String(),
			completeNumber: 0,
			total:          total,
		}
	} else {
		logrus.Warnf("%s already exist in easyProgressUtil", ip)
	}
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

// CopyR scp remote file to local
func (s *SSH) CopyR(host net.IP, localFilePath, remoteFilePath string) error {
	if utilsnet.IsLocalIP(host, s.LocalAddress) {
		if remoteFilePath != localFilePath {
			logrus.Debugf("local copy files src %s to dst %s", remoteFilePath, localFilePath)
			return osi.RecursionCopy(remoteFilePath, localFilePath)
		}
		return nil
	}
	sshClient, sftpClient, err := s.sftpConnect(host)
	if err != nil {
		return fmt.Errorf("failed to new sftp client: %v", err)
	}
	defer func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}()
	// open remote source file
	srcFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file(%s): %v", remoteFilePath, err)
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()

	err = s.Fs.MkdirAll(filepath.Dir(localFilePath))
	if err != nil {
		return err
	}
	// open local Destination file
	dstFile, err := os.Create(filepath.Clean(localFilePath))
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer func() {
		if err := dstFile.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()
	// copy to local file
	_, err = srcFile.WriteTo(dstFile)
	return err
}

// Copy file or dir to remotePath, add md5 validate
func (s *SSH) Copy(host net.IP, localPath, remotePath string) error {
	go displayInitOnce.Do(displayInit)
	if utilsnet.IsLocalIP(host, s.LocalAddress) {
		if localPath == remotePath {
			return nil
		}
		logrus.Debugf("local copy files src %s to dst %s", localPath, remotePath)
		return osi.RecursionCopy(localPath, remotePath)
	}
	logrus.Debugf("remote copy files src %s to dst %s", localPath, remotePath)
	sshClient, sftpClient, err := s.sftpConnect(host)
	if err != nil {
		return fmt.Errorf("failed to new sftp client of host(%s): %s", host, err)
	}
	defer func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}()

	f, err := s.Fs.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to get file stat of path(%s): %s", localPath, err)
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
		number = osi.CountDirFiles(localPath)
	}
	// no file in dir, do need to send
	if number == 0 {
		return nil
	}
	epu, ok := epuMap[host.String()]
	if !ok {
		registerEpu(host, number)
		epu = epuMap[host.String()]
	} else {
		epu.total += number
	}

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

func (s *SSH) remoteMd5Sum(host net.IP, remoteFilePath string) string {
	cmd := fmt.Sprintf(Md5sumCmd, remoteFilePath)
	remoteMD5, err := s.CmdToString(host, cmd, "")
	if err != nil {
		logrus.Errorf("failed to count md5 of remote file(%s) on host(%s): %v", remoteFilePath, host, err)
	}
	return strings.ReplaceAll(remoteMD5, "\r", "")
}

func (s *SSH) copyLocalDirToRemote(host net.IP, sftpClient *sftp.Client, localPath, remotePath string, epu *easyProgressUtil) {
	localFiles, err := ioutil.ReadDir(localPath)
	if err != nil {
		logrus.Errorf("failed to read local path dir(%s) on host(%s): %s", localPath, host, err)
		return
	}
	if err = sftpClient.MkdirAll(remotePath); err != nil {
		logrus.Errorf("failed to create remote path %s: %v", remotePath, err)
		return
	}
	for _, file := range localFiles {
		lfp := path.Join(localPath, file.Name())
		rfp := path.Join(remotePath, file.Name())
		if file.IsDir() {
			if err = sftpClient.MkdirAll(rfp); err != nil {
				logrus.Errorf("failed to create remote path %s: %v", rfp, err)
				return
			}
			s.copyLocalDirToRemote(host, sftpClient, lfp, rfp, epu)
		} else {
			err := s.copyLocalFileToRemote(host, sftpClient, lfp, rfp)
			if err != nil {
				errMsg := fmt.Sprintf("failed to copy local file(%s) to remote(%s) on host(%s): %v", lfp, rfp, host, err)
				epu.fail(err)
				logrus.Error(errMsg)
				return
			}
			epu.increment()
		}
	}
}

// check the remote file existence before copying
func (s *SSH) copyLocalFileToRemote(host net.IP, sftpClient *sftp.Client, localPath, remotePath string) error {
	var (
		srcMd5, dstMd5 string
	)
	srcMd5 = localMd5Sum(localPath)
	if exist, err := s.IsFileExist(host, remotePath); err != nil {
		return err
	} else if exist {
		dstMd5 = s.remoteMd5Sum(host, remotePath)
		if srcMd5 == dstMd5 {
			logrus.Debugf("remote dst %s already exists and is the latest version , skip copying process", remotePath)
			return nil
		}
	}

	srcFile, err := os.Open(filepath.Clean(localPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()

	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return err
	}
	fileStat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stat: %v", err)
	}
	// TODO seems not work
	if err := dstFile.Chmod(fileStat.Mode()); err != nil {
		return fmt.Errorf("failed to chmod remote file: %v", err)
	}
	defer func() {
		if err := dstFile.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
	}()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	dstMd5 = s.remoteMd5Sum(host, remotePath)
	if srcMd5 != dstMd5 {
		return fmt.Errorf("[ssh][%s] failed to validate md5sum: (%s != %s)", host, srcMd5, dstMd5)
	}
	return nil
}

// RemoteDirExist if remote file not exist return false and nil
func (s *SSH) RemoteDirExist(host net.IP, remoteDirPath string) (bool, error) {
	sshClient, sftpClient, err := s.sftpConnect(host)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}()
	if _, err := sftpClient.ReadDir(remoteDirPath); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SSH) IsFileExist(host net.IP, remoteFilePath string) (bool, error) {
	sshClient, sftpClient, err := s.sftpConnect(host)
	if err != nil {
		return false, fmt.Errorf("failed to new sftp client of host(%s): %s", host, err)
	}
	defer func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}()
	_, err = sftpClient.Stat(remoteFilePath)
	if err == os.ErrNotExist {
		return false, nil
	}
	return err == nil, err
}
