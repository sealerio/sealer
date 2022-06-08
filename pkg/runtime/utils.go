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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/utils/exec"
	"github.com/sirupsen/logrus"

	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sealerio/sealer/common"
	osi "github.com/sealerio/sealer/utils/os"
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
		logrus.Errorf("error version format %s %s", v1, v2)
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

func GetKubectlAndKubeconfig(ssh ssh.Interface, host, rootfs string) error {
	// fetch the cluster kubeconfig, and add /etc/hosts "EIP apiserver.cluster.local" so we can get the current cluster status later
	err := ssh.Fetch(host, path.Join(common.DefaultKubeConfigDir(), "config"), common.KubeAdminConf)
	if err != nil {
		return errors.Wrap(err, "failed to copy kubeconfig")
	}
	_, err = exec.RunSimpleCmd(fmt.Sprintf("cat /etc/hosts |grep '%s %s' || echo '%s %s' >> /etc/hosts",
		host, common.APIServerDomain, host, common.APIServerDomain))
	if err != nil {
		return errors.Wrap(err, "failed to add master IP to etc hosts")
	}

	if !osi.IsFileExist(common.KubectlPath) {
		err = osi.RecursionCopy(filepath.Join(rootfs, "bin/kubectl"), common.KubectlPath)
		if err != nil {
			return err
		}
		err = exec.Cmd("chmod", "+x", common.KubectlPath)
		if err != nil {
			return errors.Wrap(err, "chmod a+x kubectl failed")
		}
	}
	return nil
}

// LoadMetadata :read metadata via ClusterImage name.
func LoadMetadata(rootfs string) (*Metadata, error) {
	metadataPath := filepath.Join(rootfs, common.DefaultMetadataName)
	var metadataFile []byte
	var err error
	var md Metadata
	if !osi.IsFileExist(metadataPath) {
		return nil, nil
	}

	metadataFile, err = ioutil.ReadFile(filepath.Clean(metadataPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read ClusterImage metadata %v", err)
	}
	err = json.Unmarshal(metadataFile, &md)
	if err != nil {
		return nil, fmt.Errorf("failed to load ClusterImage metadata %v", err)
	}
	return &md, nil
}

func GetClusterImagePlatform(rootfs string) (cp ocispecs.Platform) {
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

func IsInContainer() bool {
	data, err := osi.NewFileReader("/proc/1/environ").ReadAll()
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "container=docker")
}
