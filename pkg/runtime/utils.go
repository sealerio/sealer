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

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kube-proxy/config/v1alpha1"
	"k8s.io/kubelet/config/v1beta1"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/pkg/runtime/kubeadm_types/v1beta2"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
	"github.com/sealerio/sealer/utils/ssh"
)

// VersionCompare :if v1 >= v2 return true, else return false
func VersionCompare(v1, v2 string) bool {
	v1 = strings.Replace(v1, "v", "", -1)
	v2 = strings.Replace(v2, "v", "", -1)
	v1 = strings.Split(v1, "-")[0]
	v2 = strings.Split(v2, "-")[0]
	v1List := strings.Split(v1, ".")
	v2List := strings.Split(v2, ".")

	if len(v1List) != 3 || len(v2List) != 3 {
		logger.Error("error version format %s %s", v1, v2)
		return false
	}
	if v1List[0] > v2List[0] {
		return true
	} else if v1List[0] < v2List[0] {
		return false
	}
	if v1List[1] > v2List[1] {
		return true
	} else if v1List[1] < v2List[1] {
		return false
	}
	if v1List[2] > v2List[2] {
		return true
	}
	return true
}

func PreInitMaster0(sshClient ssh.Interface, remoteHostIP string) error {
	err := ssh.WaitSSHReady(sshClient, 6, remoteHostIP)
	if err != nil {
		return fmt.Errorf("apply cloud cluster failed: %s", err)
	}
	// send sealer and cluster file to remote host
	sealerPath := utils.ExecutableFilePath()
	err = sshClient.Copy(remoteHostIP, sealerPath, common.RemoteSealerPath)
	if err != nil {
		return fmt.Errorf("send sealer to remote host %s failed:%v", remoteHostIP, err)
	}
	err = sshClient.CmdAsync(remoteHostIP, fmt.Sprintf(common.ChmodCmd, common.RemoteSealerPath))
	if err != nil {
		return fmt.Errorf("chmod +x sealer on remote host %s failed:%v", remoteHostIP, err)
	}
	logger.Info("send sealer cmd to %s success !", remoteHostIP)

	// send tmp cluster file
	err = sshClient.Copy(remoteHostIP, common.TmpClusterfile, common.TmpClusterfile)
	if err != nil {
		return fmt.Errorf("send cluster file to remote host %s failed:%v", remoteHostIP, err)
	}
	logger.Info("send cluster file to %s success !", remoteHostIP)

	// send register login info
	authFile := common.DefaultRegistryAuthConfigDir()
	if utils.IsFileExist(authFile) {
		err = sshClient.Copy(remoteHostIP, authFile, common.DefaultRegistryAuthDir)
		if err != nil {
			return fmt.Errorf("failed to send register config %s to remote host %s err: %v", authFile, remoteHostIP, err)
		}
		logger.Info("send register info to %s success !", remoteHostIP)
	} else {
		logger.Warn("failed to find %s, if image registry is private, please login first", authFile)
	}
	return nil
}

func GetKubectlAndKubeconfig(ssh ssh.Interface, host, rootfs string) error {
	// fetch the cluster kubeconfig, and add /etc/hosts "EIP apiserver.cluster.local" so we can get the current cluster status later
	err := ssh.Fetch(host, path.Join(common.DefaultKubeConfigDir(), "config"), common.KubeAdminConf)
	if err != nil {
		return errors.Wrap(err, "failed to copy kubeconfig")
	}
	_, err = utils.RunSimpleCmd(fmt.Sprintf("cat /etc/hosts |grep '%s %s' || echo '%s %s' >> /etc/hosts",
		host, common.APIServerDomain, host, common.APIServerDomain))
	if err != nil {
		return errors.Wrap(err, "failed to add master IP to etc hosts")
	}
	if !utils.IsFileExist(common.KubectlPath) {
		_, err = utils.CopySingleFile(filepath.Join(rootfs, "bin/kubectl"), common.KubectlPath)
		if err != nil {
			return err
		}
		err = utils.Cmd("chmod", "+x", common.KubectlPath)
		if err != nil {
			return errors.Wrap(err, "chmod a+x kubectl failed")
		}
	}
	return nil
}

// LoadMetadata :read metadata via cluster image name.
func LoadMetadata(rootfs string) (*Metadata, error) {
	metadataPath := filepath.Join(rootfs, common.DefaultMetadataName)
	var metadataFile []byte
	var err error
	var md Metadata
	if !utils.IsFileExist(metadataPath) {
		return nil, nil
	}

	metadataFile, err = ioutil.ReadFile(filepath.Clean(metadataPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read CloudImage metadata %v", err)
	}
	err = json.Unmarshal(metadataFile, &md)
	if err != nil {
		return nil, fmt.Errorf("failed to load CloudImage metadata %v", err)
	}
	return &md, nil
}

func GetCloudImagePlatform(rootfs string) (cp ocispecs.Platform) {
	// current we only support build on linux
	cp = ocispecs.Platform{
		Architecture: "amd64",
		OS:           "linux",
		Variant:      "",
		OSVersion:    "",
	}
	meta, err := LoadMetadata(rootfs)
	if err != nil {
		return
	}
	if meta == nil {
		return
	}
	if meta.Arch != "" {
		cp.Architecture = meta.Arch
	}
	if meta.Variant != "" {
		cp.Variant = meta.Variant
	}
	return
}

func DecodeCRDFromFile(filePath string, kind string) (interface{}, error) {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to dump config %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("failed to dump config close clusterfile failed %v", err)
		}
	}()
	return DecodeCRDFromReader(file, kind)
}

func DecodeCRDFromReader(r io.Reader, kind string) (interface{}, error) {
	d := yaml.NewYAMLOrJSONDecoder(r, 4096)

	for {
		ext := k8sruntime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// TODO: This needs to be able to handle object in other encodings and schemas.
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		metaType := metav1.TypeMeta{}
		err := yaml.Unmarshal(ext.Raw, &metaType)
		if err != nil {
			return nil, fmt.Errorf("decode cluster failed %v", err)
		}
		// ext.Raw
		if metaType.Kind == kind {
			return TypeConversion(ext.Raw, kind)
		}
	}
	return nil, nil
}

func DecodeCRDFromString(config string, kind string) (interface{}, error) {
	return DecodeCRDFromReader(strings.NewReader(config), kind)
}

func TypeConversion(raw []byte, kind string) (i interface{}, err error) {
	i = typeConversion(kind)
	if i == nil {
		return nil, fmt.Errorf("not found type %s from %s", kind, string(raw))
	}
	return i, yaml.Unmarshal(raw, i)
}

func typeConversion(kind string) interface{} {
	switch kind {
	case Cluster:
		return &v2.Cluster{}
	case InitConfiguration:
		return &v1beta2.InitConfiguration{}
	case JoinConfiguration:
		return &v1beta2.JoinConfiguration{}
	case ClusterConfiguration:
		return &v1beta2.ClusterConfiguration{}
	case KubeletConfiguration:
		return &v1beta1.KubeletConfiguration{}
	case KubeProxyConfiguration:
		return &v1alpha1.KubeProxyConfiguration{}
	}
	return nil
}

func RemoteCerts(altNames []string, hostIP, hostName, serviceCIRD, DNSDomain string) string {
	cmd := "seautil certs "
	if hostIP != "" {
		cmd += fmt.Sprintf(" --node-ip %s", hostIP)
	}

	if hostName != "" {
		cmd += fmt.Sprintf(" --node-name %s", hostName)
	}

	if serviceCIRD != "" {
		cmd += fmt.Sprintf(" --service-cidr %s", serviceCIRD)
	}

	if DNSDomain != "" {
		cmd += fmt.Sprintf(" --dns-domain %s", DNSDomain)
	}

	for _, name := range append(altNames, common.APIServerDomain) {
		if name != "" {
			cmd += fmt.Sprintf(" --alt-names %s", name)
		}
	}

	return cmd
}
