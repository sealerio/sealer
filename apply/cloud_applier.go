package apply

import (
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/guest"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/infra"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ApplyCluster = "chmod +x %s && %s apply -f %s"

type CloudApplier struct {
	*DefaultApplier
}

func NewAliCloudProvider(cluster *v1.Cluster) Interface {
	d := &DefaultApplier{
		ClusterDesired: cluster,
		ImageManager:   image.NewImageService(),
		FileSystem:     filesystem.NewFilesystem(),
		Runtime:        runtime.NewDefaultRuntime(cluster),
		Guest:          guest.NewGuestManager(),
	}
	return &CloudApplier{d}
}

func (c *CloudApplier) Apply() error {
	cluster := c.ClusterDesired
	cloudProvider := infra.NewDefaultProvider(cluster)
	if cloudProvider == nil {
		return fmt.Errorf("new cloud provider failed")
	}
	err := cloudProvider.Apply()
	if err != nil {
		return fmt.Errorf("apply infra failed %v", err)
	}
	if cluster.DeletionTimestamp != nil {
		return nil
	}
	err = c.SaveClusterfile()
	if err != nil {
		return err
	}
	cluster.Spec.Provider = common.BAREMETAL
	err = utils.MarshalYamlToFile(common.TmpClusterfile, cluster)
	if err != nil {
		return fmt.Errorf("marshal tmp cluster file failed %v", err)
	}
	defer func() {
		if err := utils.CleanFiles(common.TmpClusterfile); err != nil {
			logger.Error("failed to clean %s, err: %v", common.TmpClusterfile, err)
		}
	}()
	client, err := ssh.NewSSHClientWithCluster(cluster)
	if err != nil {
		return fmt.Errorf("prepare cluster ssh client failed %v", err)
	}

	err = runtime.PreInitMaster0(client.SSH, client.Host)
	if err != nil {
		return err
	}
	err = client.SSH.CmdAsync(client.Host, fmt.Sprintf(ApplyCluster, common.RemoteSealerPath, common.RemoteSealerPath, common.TmpClusterfile))
	if err != nil {
		return err
	}
	// fetch the cluster kubeconfig, and add /etc/hosts "EIP apiserver.cluster.local" so we can get the current cluster status later
	err = client.SSH.Fetch(client.Host, common.DefaultKubeconfig, common.KubeAdminConf)
	if err != nil {
		return err
	}
	err = utils.AppendFile(common.EtcHosts, fmt.Sprintf("%s %s", client.Host, common.APIServerDomain))
	if err != nil {
		return errors.Wrap(err, "append EIP to etc hosts failed")
	}
	err = client.SSH.Fetch(client.Host, common.KubectlPath, common.KubectlPath)
	if err != nil {
		return errors.Wrap(err, "fetch kubectl failed")
	}
	err = utils.Cmd("chmod", "+x", common.KubectlPath)

	if err != nil {
		return errors.Wrap(err, "add EIP to etc hosts failed")
	}
	return nil
}

func (c *CloudApplier) Delete() error {
	t := metav1.Now()
	c.ClusterDesired.DeletionTimestamp = &t
	host := c.ClusterDesired.GetAnnotationsByKey(common.Eip)
	err := c.Apply()
	if err != nil {
		return err
	}
	if err := utils.RemoveFileContent(common.EtcHosts, fmt.Sprintf("%s %s", host, common.APIServerDomain)); err != nil {
		logger.Warn(err)
	}

	if err := utils.CleanFiles(common.DefaultKubeconfigDir, common.GetClusterWorkDir(c.ClusterDesired.Name), common.TmpClusterfile, common.KubectlPath); err != nil {
		logger.Warn(err)
		return nil
	}

	return nil
}

func (c *CloudApplier) SaveClusterfile() error {
	fileName := common.GetClusterWorkClusterfile(c.ClusterDesired.Name)
	err := utils.MkFileFullPathDir(fileName)
	if err != nil {
		return fmt.Errorf("mkdir failed %s %v", fileName, err)
	}
	err = utils.MarshalYamlToFile(fileName, c.ClusterDesired)
	if err != nil {
		return fmt.Errorf("marshal cluster file failed %v", err)
	}
	return nil
}
