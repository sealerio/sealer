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

	"github.com/alibaba/sealer/utils/progress"

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

func (s *SSH) RemoteMd5Sum(host, remoteFilePath string) string {
	cmd := fmt.Sprintf(Md5sumCmd, remoteFilePath)
	remoteMD5, err := s.CmdToString(host, cmd, "")
	if err != nil {
		logger.Error("count remote md5 failed %s %s", host, remoteFilePath, err)
	}
	return remoteMD5
}

//CmdToString is in host exec cmd and replace to spilt str
func (s *SSH) CmdToString(host, cmd, spilt string) (string, error) {
	data, err := s.Cmd(host, cmd)
	if err != nil {
		return "", fmt.Errorf("exec remote command failed %s %s %s", host, cmd, err)
	}
	if data != nil {
		str := string(data)
		str = strings.ReplaceAll(str, "\r\n", spilt)
		str = strings.ReplaceAll(str, "\n", spilt)
		return str, nil
	}
	return "", fmt.Errorf("command %s %s return nil", host, cmd)
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
	addr = s.addrReformat(host)

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
	logger.Debug("copy files src %s to dst %s", localPath, remotePath)
	baseRemoteFilePath := filepath.Dir(remotePath)
	mkDstDir := fmt.Sprintf("mkdir -p %s || true", baseRemoteFilePath)
	err := s.CmdAsync(host, mkDstDir)
	if err != nil {
		return err
	}
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

	number := 1
	if f.IsDir() {
		number = utils.CountDirFiles(localPath)
	}
	// no file in dir, do need to send
	if number == 0 {
		return nil
	}

	ch := make(chan progress.Msg, 10)
	defer close(ch)
	flow := progress.NewProgressFlow()
	flow.AddProgressTasks(progress.TaskDef{
		Task: "Copying Files",
		Max:  int64(number),
		ProgressSrc: progress.ChannelTask{
			ProgressChan: ch,
		},
		SuccessMsg: fmt.Sprintf("Success to copy %s to %s", localPath, remotePath),
		FailMsg:    fmt.Sprintf("Failed to copy %s to %s", localPath, remotePath),
	})

	go func() {
		if f.IsDir() {
			s.copyLocalDirToRemote(host, sshClient, sftpClient, localPath, remotePath, ch)
		} else {
			err = s.copyLocalFileToRemote(host, sshClient, sftpClient, localPath, remotePath)
			if err != nil {
				ch <- progress.Msg{Status: progress.StatusFail}
			}
			ch <- progress.Msg{Inc: 1}
		}
	}()
	flow.Start()
	if err != nil {
		return err
	}
	return nil
}

func (s *SSH) copyLocalDirToRemote(host string, sshClient *ssh.Client, sftpClient *sftp.Client, localPath, remotePath string, ch chan progress.Msg) {
	localFiles, err := ioutil.ReadDir(localPath)
	if err != nil {
		logger.Error("read local path dir failed %s %s", host, localPath)
		return
	}
	sftpClient.Mkdir(remotePath)
	for _, file := range localFiles {
		lfp := path.Join(localPath, file.Name())
		rfp := path.Join(remotePath, file.Name())
		if file.IsDir() {
			sftpClient.Mkdir(rfp)
			s.copyLocalDirToRemote(host, sshClient, sftpClient, lfp, rfp, ch)
		} else {
			err := s.copyLocalFileToRemote(host, sshClient, sftpClient, lfp, rfp)
			if err != nil {
				errMsg := fmt.Sprintf("copy local file to remote failed %v %s %s %s", err, host, lfp, rfp)
				ch <- progress.Msg{Status: progress.StatusFail, Msg: errMsg}
				logger.Error(errMsg)
				return
			}
			ch <- progress.Msg{Inc: 1}
		}
	}
}

// solve the sesion
func (s *SSH) copyLocalFileToRemote(host string, sshClient *ssh.Client, sftpClient *sftp.Client, localPath, remotePath string) error {
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
	srcMd5 := FromLocal(localPath)
	dstMd5 := s.RemoteMd5Sum(host, remotePath)
	if srcMd5 != dstMd5 {
		return fmt.Errorf("[ssh][%s] validate md5sum failed %s != %s", host, srcMd5, dstMd5)
	}
	return nil
}

func FromLocal(localPath string) string {
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
		return false, nil
	}
	return true, nil
}
