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
	"fmt"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
)

// Overwrite templete filename
const (
	bootstrapTemplateName       = "kubeadm-bootstrap.yaml.tmpl"
	initConfigTemplateName      = "kubeadm-init.yaml.tmpl"
	clusterConfigTemplateName   = "kubeadm-cluster-config.yaml.tmpl"
	kubeproxyConfigTemplateName = "kubeadm-kubeproxy-config.yaml.tmpl"
	kubeletConfigTemplateName   = "kubeadm-kubelet-config.yaml.tmpl"
	joinConfigTemplateName      = "kubeadm-join-config.yaml.tmpl"
)

const ( /* #nosec G101  */
	bootstrapTokenDefault = `apiVersion: {{.KubeadmAPI}}
caCertPath: /etc/kubernetes/pki/ca.crt
discovery:
  bootstrapToken:
    {{- if .Master}}
    apiServerEndpoint: {{.Master0}}:6443
    {{else}}
    apiServerEndpoint: {{.VIP}}:6443
    {{end -}}
    token: {{.TokenDiscovery}}
    caCertHashes:
    - {{.TokenDiscoveryCAHash}}
  timeout: 5m0s
`
	initConfigurationDefault = `apiVersion: {{.KubeadmAPI}}
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: {{.Master0}}
  bindPort: 6443
nodeRegistration:
  criSocket: {{.CriSocket}}
`

	joinConfigurationDefault = `kind: JoinConfiguration
{{- if .Master }}
controlPlane:
  localAPIEndpoint:
    advertiseAddress: {{.Master}}
    bindPort: 6443
{{- end}}
nodeRegistration:
  criSocket: {{.CriSocket}}
`

	clusterConfigurationDefault = `apiVersion: {{.KubeadmAPI}}
kind: ClusterConfiguration
kubernetesVersion: {{.Version}}
controlPlaneEndpoint: "{{.ApiServer}}:6443"
imageRepository: {{.Repo}}
networking:
  # dnsDomain: cluster.local
  podSubnet: {{.PodCIDR}}
  serviceSubnet: {{.SvcCIDR}}
apiServer:
  certSANs:
  {{range .CertSANS -}}
  - {{.}}
  {{end -}}
  extraArgs:
    etcd-servers: {{.EtcdServers}}
    feature-gates: TTLAfterFinished=true,EphemeralContainers=true
    audit-policy-file: "/etc/kubernetes/audit-policy.yml"
    audit-log-path: "/var/log/kubernetes/audit.log"
    audit-log-format: json
    audit-log-maxbackup: '"10"'
    audit-log-maxsize: '"100"'
    audit-log-maxage: '"7"'
    enable-aggregator-routing: '"true"'
  extraVolumes:
  - name: "audit"
    hostPath: "/etc/kubernetes"
    mountPath: "/etc/kubernetes"
    pathType: DirectoryOrCreate
  - name: "audit-log"
    hostPath: "/var/log/kubernetes"
    mountPath: "/var/log/kubernetes"
    pathType: DirectoryOrCreate
  - name: localtime
    hostPath: /etc/localtime
    mountPath: /etc/localtime
    readOnly: true
    pathType: File
controllerManager:
  extraArgs:
    feature-gates: TTLAfterFinished=true,EphemeralContainers=true
    experimental-cluster-signing-duration: 876000h
  extraVolumes:
  - hostPath: /etc/localtime
    mountPath: /etc/localtime
    name: localtime
    readOnly: true
    pathType: File
scheduler:
  extraArgs:
    feature-gates: TTLAfterFinished=true,EphemeralContainers=true
  extraVolumes:
  - hostPath: /etc/localtime
    mountPath: /etc/localtime
    name: localtime
    readOnly: true
    pathType: File
etcd:
  local:
    extraArgs:
      listen-metrics-urls: http://0.0.0.0:2381
`
	kubeproxyConfigDefault = `apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  excludeCIDRs:
  - "{{.VIP}}/32"
`

	kubeletConfigDefault = `apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
authentication:
  anonymous:
    enabled: false
  webhook:
    cacheTTL: 2m0s
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
  webhook:
    cacheAuthorizedTTL: 5m0s
    cacheUnauthorizedTTL: 30s
cgroupDriver: {{ .CgroupDriver}}
cgroupsPerQOS: true
clusterDomain: cluster.local
configMapAndSecretChangeDetectionStrategy: Watch
containerLogMaxFiles: 5
containerLogMaxSize: 10Mi
contentType: application/vnd.kubernetes.protobuf
cpuCFSQuota: true
cpuCFSQuotaPeriod: 100ms
cpuManagerPolicy: none
cpuManagerReconcilePeriod: 10s
enableControllerAttachDetach: true
enableDebuggingHandlers: true
enforceNodeAllocatable:
- pods
eventBurst: 10
eventRecordQPS: 5
evictionHard:
  imagefs.available: 15%
  memory.available: 100Mi
  nodefs.available: 10%
  nodefs.inodesFree: 5%
evictionPressureTransitionPeriod: 5m0s
failSwapOn: true
fileCheckFrequency: 20s
hairpinMode: promiscuous-bridge
healthzBindAddress: 127.0.0.1
healthzPort: 10248
httpCheckFrequency: 20s
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 80
imageMinimumGCAge: 2m0s
iptablesDropBit: 15
iptablesMasqueradeBit: 14
kubeAPIBurst: 10
kubeAPIQPS: 5
makeIPTablesUtilChains: true
maxOpenFiles: 1000000
maxPods: 110
nodeLeaseDurationSeconds: 40
nodeStatusReportFrequency: 10s
nodeStatusUpdateFrequency: 10s
oomScoreAdj: -999
podPidsLimit: -1
port: 10250
registryBurst: 10
registryPullQPS: 5
rotateCertificates: true
runtimeRequestTimeout: 2m0s
serializeImagePulls: true
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 4h0m0s
syncFrequency: 1m0s
volumeStatsAggPeriod: 1m0s`
	ContainerdShell = `if grep "SystemdCgroup = true"  /etc/containerd/config.toml &> /dev/null; then  
driver=systemd
else
driver=cgroupfs
fi
echo ${driver}`
	DockerShell = `driver=$(docker info -f "{{.CgroupDriver}}")
	echo "${driver}"`
)

// Get from rootfs or by default
func getInitTemplateText(clusterName string) string {
	return fmt.Sprintf("%s\n---\n%s\n---\n%s\n---\n%s",
		readFromFileOrDefault(clusterName, initConfigTemplateName, initConfigurationDefault),
		readFromFileOrDefault(clusterName, clusterConfigTemplateName, clusterConfigurationDefault),
		readFromFileOrDefault(clusterName, kubeproxyConfigTemplateName, kubeproxyConfigDefault),
		readFromFileOrDefault(clusterName, kubeletConfigTemplateName, kubeletConfigDefault),
	)
}

func getJoinTemplateText(clusterName string) string {
	return fmt.Sprintf("%s\n%s\n---\n%s",
		readFromFileOrDefault(clusterName, bootstrapTemplateName, bootstrapTokenDefault),
		readFromFileOrDefault(clusterName, joinConfigTemplateName, joinConfigurationDefault),
		readFromFileOrDefault(clusterName, kubeletConfigTemplateName, kubeletConfigDefault),
	)
}

// return file content or defaultContent if file not exist
func readFromFileOrDefault(clusterName, fileName, defaultContent string) string {
	fileName = filepath.Join(common.DefaultMountCloudImageDir(clusterName), common.EtcDir, fileName)
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return defaultContent
	}
	if err != nil {
		logger.Warn("kubeadm template file stat %v", err)
		return defaultContent
	}

	bs, err := utils.ReadAll(fileName)
	if err != nil {
		logger.Warn("read kubeadm template file err %v", err)
		return defaultContent
	}
	return string(bs)
}
