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
	"fmt"
	"html/template"
	"strings"

	"github.com/alibaba/sealer/utils"
	"sigs.k8s.io/yaml"
)
//Get default sans
func (d *Default) getDefaultSANs() []string {
	var sans = []string{"127.0.0.1", "apiserver.cluster.local", d.VIP}
	// append specified certSANS
	sans = append(sans, d.APIServerCertSANs...)
	// append all k8s master node ip
	sans = append(sans, utils.GetHostIPSlice(d.Masters)...)
	return sans
}

//Template is
func (d *Default) defaultTemplate() ([]byte, error) {
	return d.templateFromContent(d.kubeadmConfig())
}

func (d *Default) templateFromContent(templateContent string) ([]byte, error) {
	tmpl, err := template.New("text").Parse(templateContent)
	if err != nil {
		return nil, err
	}

	var envMap = make(map[string]interface{})
	sans := []string{"127.0.0.1"}
	sans = utils.AppendIPList(sans, []string{d.APIServer})
	sans = utils.AppendIPList(sans, utils.GetHostIPSlice(d.Masters))
	sans = utils.AppendIPList(sans, d.APIServerCertSANs)
	sans = utils.AppendIPList(sans, []string{d.VIP})

	envMap[CertSANS] = sans
	envMap[VIP] = d.VIP
	envMap[Masters] = utils.GetHostIPSlice(d.Masters)
	envMap[Version] = d.Metadata.Version
	envMap[APIServer] = d.APIServer
	envMap[PodCIDR] = d.PodCIDR
	envMap[SvcCIDR] = d.SvcCIDR
	envMap[Repo] = fmt.Sprintf("%s:%d/library", SeaHub, d.RegistryPort)
	envMap[EtcdServers] = getEtcdEndpointsWithHTTPSPrefix(d.Masters)
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, envMap)
	return buffer.Bytes(), err
}

func (d *Default) kubeadmConfig() string {
	var sb strings.Builder
	// kubernetes gt 1.20, use Containerd instead of docker
	if VersionCompare(d.Metadata.Version, V1200) {
		sb.Write([]byte(InitTemplateTextV1bate2))
	} else {
		sb.Write([]byte(InitTemplateTextV1beta1))
	}
	return sb.String()
}

//yaml data unmarshal kubeadm type struct
func kubeadmDataFromYaml(context string) *kubeadmType {
	yamls := strings.Split(context, "---")
	if len(yamls) <= 0 {
		return nil
	}
	for _, y := range yamls {
		cfg := strings.TrimSpace(y)
		if cfg == "" {
			continue
		}
		kubeadm := &kubeadmType{}
		if err := yaml.Unmarshal([]byte(cfg), kubeadm); err != nil {
			//TODO, solve error?
			continue
		}
		if kubeadm.Kind != "ClusterConfiguration" {
			continue
		}
		if kubeadm.Networking.DNSDomain == "" {
			kubeadm.Networking.DNSDomain = "cluster.local"
		}
		return kubeadm
	}
	return nil
}

type kubeadmType struct {
	Kind      string `yaml:"kind,omitempty"`
	APIServer struct {
		CertSANs []string `yaml:"certSANs,omitempty"`
	} `yaml:"apiServer"`
	Networking struct {
		DNSDomain string `yaml:"dnsDomain,omitempty"`
	} `yaml:"networking"`
}

func getEtcdEndpointsWithHTTPSPrefix(masters []string) string {
	var tmpSlice []string
	for _, ip := range masters {
		tmpSlice = append(tmpSlice, fmt.Sprintf("https://%s:2379", utils.GetHostIP(ip)))
	}
	return strings.Join(tmpSlice, ",")
}
