package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	imageUtils "github.com/alibaba/sealer/image/utils"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/mount"
	"github.com/alibaba/sealer/utils/ssh"
)

const (
	RemoteChmod = "cd %s  && chmod +x scripts/* && cd scripts && sh init.sh"
)

type Interface interface {
	Mount(cluster *v1.Cluster, hosts []string) error
	UnMount(cluster *v1.Cluster) error
}
type FileSystem struct {
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func (f *FileSystem) mountImage(cluster *v1.Cluster) error {
	clusterTmpRootfsDir := filepath.Join("/tmp", cluster.Name)
	if IsDir(clusterTmpRootfsDir) {
		logger.Info("cluster rootfs already exist, skip mount cluster image")
		return nil
	}
	//get layers
	Image, err := imageUtils.GetImage(cluster.Spec.Image)
	if err != nil {
		return err
	}
	logger.Info("image name is %s", Image.Name)
	layers, err := image.GetImageLayerDirs(Image)
	if err != nil {
		return fmt.Errorf("get layers failed: %v", err)
	}
	driver := mount.NewMountDriver()
	upperDir := filepath.Join(clusterTmpRootfsDir, "upper")
	if err = os.MkdirAll(upperDir, 0744); err != nil {
		return fmt.Errorf("create upperdir failed, %s", err)
	}
	if err = driver.Mount(clusterTmpRootfsDir, upperDir, layers...); err != nil {
		return fmt.Errorf("mount files failed %v", err)
	}
	return nil
}

func (f *FileSystem) Mount(cluster *v1.Cluster, hosts []string) error {
	err := f.mountImage(cluster)
	if err != nil {
		return err
	}

	clusterRootfsDir := filepath.Join(common.DefaultClusterRootfsDir, cluster.Name)
	//scp roofs to all Masters and Nodes,then do init.sh
	if err = mountRootfs(hosts, clusterRootfsDir, cluster); err != nil {
		return fmt.Errorf("mount rootfs failed %v", err)
	}
	return nil
}

func (f *FileSystem) UnMount(cluster *v1.Cluster) error {
	//do clean.sh,then remove all Masters and Nodes roofs
	IPList := append(cluster.Spec.Masters.IPList, cluster.Spec.Nodes.IPList...)
	if err := unmountRootfs(IPList, cluster); err != nil {
		return err
	}
	return nil
}

func mountRootfs(ipList []string, target string, cluster *v1.Cluster) error {
	SSH := ssh.NewSSHByCluster(cluster)
	if err := ssh.WaitSSHReady(SSH, ipList...); err != nil {
		return errors.Wrap(err, "check for node ssh service time out")
	}
	var wg sync.WaitGroup
	var flag bool
	var mutex sync.Mutex
	rootfs := filepath.Join(common.DefaultClusterRootfsDir, cluster.Name)
	src := filepath.Join("/tmp", cluster.Name)
	// TODO scp sdk has change file mod bug
	initCmd := fmt.Sprintf(RemoteChmod, rootfs)
	for _, ip := range ipList {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			err := SSH.Copy(ip, src, target)
			if err != nil {
				logger.Error("copy rootfs failed %v", err)
				mutex.Lock()
				flag = true
				mutex.Unlock()
			}
			err = SSH.CmdAsync(ip, initCmd)
			if err != nil {
				logger.Error("exec init.sh failed %v", err)
				mutex.Lock()
				flag = true
				mutex.Unlock()
			}
		}(ip)
	}
	wg.Wait()
	if flag {
		return fmt.Errorf("mountRootfs failed")
	}
	return nil
}

func unmountRootfs(ipList []string, cluster *v1.Cluster) error {
	SSH := ssh.NewSSHByCluster(cluster)
	var wg sync.WaitGroup
	var flag bool
	var mutex sync.Mutex
	clusterRootfsDir := filepath.Join(common.DefaultClusterRootfsDir, cluster.Name)
	execClean := fmt.Sprintf("/bin/sh -c "+common.DefaultClusterClearFile, cluster.Name)
	rmRootfs := fmt.Sprintf("rm -rf %s", clusterRootfsDir)
	for _, ip := range ipList {
		wg.Add(1)
		go func(IP string) {
			defer wg.Done()
			if err := SSH.CmdAsync(IP, execClean, rmRootfs); err != nil {
				logger.Error("%s:exec %s failed, %s", IP, execClean, err)
				mutex.Lock()
				flag = true
				mutex.Unlock()
				return
			}
		}(ip)
	}
	wg.Wait()
	if flag {
		return fmt.Errorf("unmountRootfs failed")
	}
	return nil
}

func NewFilesystem() Interface {
	return &FileSystem{}
}
