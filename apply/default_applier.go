package apply

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/guest"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// cloud builder using cloud provider to build a cluster image
type DefaultApplier struct {
	ClusterDesired  *v1.Cluster
	ClusterCurrent  *v1.Cluster
	ImageManager    image.Service
	FileSystem      filesystem.Interface
	Runtime         runtime.Interface
	Guest           guest.Interface
	MastersToJoin   []string
	MastersToDelete []string
	NodesToJoin     []string
	NodesToDelete   []string
}

type ActionName string

const (
	PullIfNotExist ActionName = "PullIfNotExist"
	Mount          ActionName = "Mount"
	UnMount        ActionName = "UnMount"
	Init           ActionName = "Init"
	Upgrade        ActionName = "Upgrade"
	ApplyMasters   ActionName = "ApplyMasters"
	ApplyNodes     ActionName = "ApplyNodes"
	Guest          ActionName = "Guest"
	CNI            ActionName = "CNI"
	Reset          ActionName = "Reset"
)

var ActionFuncMap = map[ActionName]func(*DefaultApplier) error{
	PullIfNotExist: func(applier *DefaultApplier) error {
		imageName := applier.ClusterDesired.Spec.Image
		return image.NewImageService().PullIfNotExist(imageName)
	},
	Mount: func(applier *DefaultApplier) error {
		// TODO mount only mount desired hosts, some hosts already mounted when update cluster
		var hosts []string
		if applier.ClusterCurrent == nil {
			hosts = append(applier.ClusterDesired.Spec.Masters.IPList, applier.ClusterDesired.Spec.Nodes.IPList...)
		} else {
			hosts = append(applier.MastersToJoin, applier.NodesToJoin...)
		}
		return applier.FileSystem.Mount(applier.ClusterDesired, hosts)
	},
	UnMount: func(applier *DefaultApplier) error {
		return applier.FileSystem.UnMount(applier.ClusterDesired)
	},
	Init: func(applier *DefaultApplier) error {
		return applier.Runtime.Init(applier.ClusterDesired)
	},
	Upgrade: func(applier *DefaultApplier) error {
		return applier.Runtime.Upgrade(applier.ClusterDesired)
	},
	ApplyMasters: func(applier *DefaultApplier) error {
		return applyMasters(applier)
	},
	ApplyNodes: func(applier *DefaultApplier) error {
		return applyNodes(applier)
	},
	Guest: func(applier *DefaultApplier) error {
		return applier.Guest.Apply(applier.ClusterDesired)
	},
	CNI: func(applier *DefaultApplier) error {
		return applier.Runtime.CNI(applier.ClusterDesired)
	},
	Reset: func(applier *DefaultApplier) error {
		return applier.Runtime.Reset(applier.ClusterDesired)
	},
}

func applyMasters(applier *DefaultApplier) error {
	err := applier.Runtime.JoinMasters(applier.MastersToJoin)
	if err != nil {
		return err
	}
	err = applier.Runtime.DeleteMasters(applier.MastersToDelete)
	if err != nil {
		return err
	}
	return nil
}

func applyNodes(applier *DefaultApplier) error {
	err := applier.Runtime.JoinNodes(applier.NodesToJoin)
	if err != nil {
		return err
	}
	err = applier.Runtime.DeleteNodes(applier.NodesToDelete)
	if err != nil {
		return err
	}
	return nil
}

func (c *DefaultApplier) Apply() (err error) {
	if c.ClusterDesired.GetDeletionTimestamp().IsZero() {
		err = saveClusterfile(c.ClusterDesired)
		if err != nil {
			return err
		}
	}

	currentCluster, err := GetCurrentCluster()
	if err != nil {
		return errors.Wrap(err, "get current cluster failed")
	}
	if currentCluster != nil {
		c.ClusterCurrent = c.ClusterDesired.DeepCopy()
		c.ClusterCurrent.Spec.Masters = currentCluster.Spec.Masters
		c.ClusterCurrent.Spec.Nodes = currentCluster.Spec.Nodes
	}
	logger.Debug("sealer apply add master0 host process")
	c.addMaster0Host(c.ClusterDesired)
	todoList, _ := c.diff()
	for _, action := range todoList {
		logger.Debug("sealer apply process %s", action)
		err = ActionFuncMap[action](c)
		if err != nil {
			return err
		}
	}
	logger.Debug("sealer apply delete master0 process")
	c.deleteMaster0Host(c.ClusterDesired)
	logger.Debug("sealer apply copy kube config process")
	c.copyKubeConfig(c.ClusterDesired)
	logger.Debug("sealer apply copy kubectl process")
	c.copyKubectl(c.ClusterDesired)
	logger.Debug("sealer apply umount rootfs process")
	c.umountRootfs(c.ClusterDesired)
	return nil
}

func (d *DefaultApplier) addMaster0Host(cluster *v1.Cluster) {
	err := utils.RemoveFileContent(common.EtcHosts, fmt.Sprintf("%s %s", cluster.Spec.Masters.IPList[0], common.APIServerDomain))
	if err != nil {
		logger.Info("remove /etc/host failed: %v", err)
	}
}
func (d *DefaultApplier) deleteMaster0Host(cluster *v1.Cluster) {
	content, err := utils.ReadAll(common.EtcHosts)
	if err != nil {
		return
	}
	if !strings.Contains(string(content), common.APIServerDomain) {
		err = utils.AppendFile(common.EtcHosts, fmt.Sprintf("%s %s", cluster.Spec.Masters.IPList[0], common.APIServerDomain))
		if err != nil {
			logger.Warn("append desired cluster new masters to etc hosts failed: %v", err)
		}
	}
}
func (d *DefaultApplier) copyKubeConfig(cluster *v1.Cluster) {
	if !utils.IsFileExist(common.DefaultKubeconfig) {
		adminConf := path.Join(common.DefaultClusterRootfsDir, cluster.Name, "admin.conf")
		if !utils.IsFileExist(adminConf) {
			adminConf = common.KubeAdminConf
		}
		_, err := utils.CopySingleFile(adminConf, common.DefaultKubeconfig)
		if err != nil {
			logger.Warn("copy kube config failed: %v", err)
		}
	}
}
func (d *DefaultApplier) copyKubectl(cluster *v1.Cluster) {
	if !utils.IsFileExist(common.KubectlPath) {
		clusterTmpRootfsDir := filepath.Join("/tmp", cluster.Name)
		kubectl := filepath.Join(clusterTmpRootfsDir, "bin", "kubectl")
		if utils.IsFileExist(kubectl) {
			_, err := utils.CopySingleFile(kubectl, common.KubectlPath)
			if err != nil {
				logger.Warn("copy kube config failed: %v", err)
			}
			err = utils.Cmd("chmod", "+x", common.KubectlPath)
			if err != nil {
				logger.Warn("chmod a+x kubectl failed: %v", err)
			}
		}
	}
}

func (c *DefaultApplier) umountRootfs(cluster *v1.Cluster) {
	clusterTmpRootfsDir := filepath.Join("/tmp", cluster.Name)
	if utils.IsFileExist(clusterTmpRootfsDir) {
		logger.Debug("unmount cluster dir %s", clusterTmpRootfsDir)
		if err := mount.NewMountDriver().Unmount(clusterTmpRootfsDir); err != nil {
			logger.Warn("failed to umount %s, err: %v", clusterTmpRootfsDir, err)
		}
	}
	utils.CleanDir(clusterTmpRootfsDir)
}

func (c *DefaultApplier) Delete() (err error) {
	t := metav1.Now()
	c.ClusterDesired.DeletionTimestamp = &t
	defer func() {
		if err = utils.CleanFiles(common.DefaultKubeconfigDir, common.GetClusterWorkDir(c.ClusterDesired.Name), common.TmpClusterfile, common.KubectlPath); err != nil {
			logger.Warn(err)
		}
		utils.CleanDir(common.GetClusterWorkDir(c.ClusterDesired.Name))
		utils.CleanDir(common.GetClusterRootfsDir(c.ClusterDesired.Name))
	}()
	return c.Apply()
}
func (c *DefaultApplier) diff() (todoList []ActionName, err error) {
	if c.ClusterDesired.DeletionTimestamp != nil {
		c.MastersToDelete = c.ClusterDesired.Spec.Masters.IPList
		c.NodesToDelete = c.ClusterDesired.Spec.Nodes.IPList
		todoList = append(todoList, Reset)
		todoList = append(todoList, UnMount)
		return todoList, nil
	}

	if c.ClusterCurrent == nil {
		todoList = append(todoList, PullIfNotExist)
		todoList = append(todoList, Mount)
		todoList = append(todoList, Init)
		c.MastersToJoin = c.ClusterDesired.Spec.Masters.IPList[1:]
		c.NodesToJoin = c.ClusterDesired.Spec.Nodes.IPList
		todoList = append(todoList, ApplyNodes)
		todoList = append(todoList, ApplyMasters)
		todoList = append(todoList, Guest)
		return todoList, nil
	}

	todoList = append(todoList, PullIfNotExist)
	if c.ClusterDesired.Spec.Image != c.ClusterCurrent.Spec.Image {
		logger.Info("current image is : %s and desired iamge is : %s , so upgrade your cluster", c.ClusterCurrent.Spec.Image, c.ClusterDesired.Spec.Image)
		todoList = append(todoList, Upgrade)
	}
	c.MastersToJoin, c.MastersToDelete = utils.GetDiffHosts(c.ClusterCurrent.Spec.Masters, c.ClusterDesired.Spec.Masters)
	c.NodesToJoin, c.NodesToDelete = utils.GetDiffHosts(c.ClusterCurrent.Spec.Nodes, c.ClusterDesired.Spec.Nodes)
	todoList = append(todoList, Mount)
	if c.NodesToJoin != nil || c.NodesToDelete != nil {
		todoList = append(todoList, ApplyNodes)
	}
	if c.MastersToJoin != nil || c.MastersToDelete != nil {
		todoList = append(todoList, ApplyMasters)
	}
	// if only contains PullIfNotExist and Mount, we do nothing
	if len(todoList) == 2 {
		todoList = []ActionName{}
	} else {
		todoList = append(todoList, Guest)
	}
	todoList = append(todoList, CNI)
	return todoList, nil
}

func NewDefaultApplier(cluster *v1.Cluster) Interface {
	return &DefaultApplier{
		ClusterDesired: cluster,
		ImageManager:   image.NewImageService(),
		FileSystem:     filesystem.NewFilesystem(),
		Runtime:        runtime.NewDefaultRuntime(cluster),
		Guest:          guest.NewGuestManager(),
	}
}
