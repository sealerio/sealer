package ssh

import (
	"fmt"
	"github.com/alibaba/sealer/common"
	"sync"
	"time"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

type Interface interface {
	// copy local files to remote host
	// scp -r /tmp root@192.168.0.2:/root/tmp => Copy("192.168.0.2","tmp","/root/tmp")
	// need check md5sum
	Copy(host, srcFilePath, dstFilePath string) error
	// copy remote host files to localhost
	Fetch(host, srcFilePath, dstFilePath string) error
	// exec command on remote host, and asynchronous return logs
	CmdAsync(host string, cmd ...string) error
	// exec command on remote host, and return combined standard output and standard error
	Cmd(host, cmd string) ([]byte, error)
	// check remote file exist or not
	IsFileExist(host, remoteFilePath string) bool
	//Remote file existence returns true, nil
	RemoteDirExist(host, remoteDirpath string) (bool, error)
	// exec command on remote host, and return spilt standard output and standard error
	CmdToString(host, cmd, spilt string) (string, error)
	Ping(host string) error
}

type SSH struct {
	User       string
	Password   string
	PkFile     string
	PkPassword string
	Timeout    *time.Duration
}

func NewSSHByCluster(cluster *v1.Cluster) Interface {
	return &SSH{
		User:       cluster.Spec.SSH.User,
		Password:   cluster.Spec.SSH.Passwd,
		PkFile:     cluster.Spec.SSH.Pk,
		PkPassword: cluster.Spec.SSH.PkPasswd,
	}
}

type SSHClient struct {
	Ssh  Interface
	Host string
}

func NewSSHClientWithCluster(cluster *v1.Cluster) (*SSHClient, error) {
	sshClient := NewSSHByCluster(cluster)
	if sshClient == nil {
		return nil, fmt.Errorf("cloud build init ssh client failed")
	}
	host := cluster.GetAnnotationsByKey(common.Eip)
	if host == "" {
		return nil, fmt.Errorf("get cluster EIP failed")
	}
	err := WaitSSHReady(sshClient, host)
	if err != nil {
		return nil, err
	}
	return &SSHClient{
		Ssh:  sshClient,
		Host: host,
	}, nil
}

func WaitSSHReady(ssh Interface, hosts ...string) error {
	var err error
	var wg sync.WaitGroup
	for _, h := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				if err := ssh.Ping(host); err == nil {
					return
				}
				time.Sleep(time.Duration(i) * time.Second)
			}
			err = fmt.Errorf("wait for [%s] ssh ready timeout", host)
		}(h)
	}
	wg.Wait()
	return err
}
