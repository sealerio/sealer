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

const InitTemplateTextV1beta1 = `
apiVersion: kubeadm.k8s.io/v1beta1
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
  - 127.0.0.1
  - {{.ApiServer}}
  {{range .Masters -}}
  - {{.}}
  {{end -}}
  {{range .CertSANS -}}
  - {{.}}
  {{end -}}
  - {{.VIP}}
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

const JoinCPTemplateTextV1beta2 = string(`apiVersion: kubeadm.k8s.io/v1beta2
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
kind: JoinConfiguration
{{- if .Master }}
controlPlane:
  localAPIEndpoint:
    advertiseAddress: {{.Master}}
    bindPort: 6443
{{- end}}
nodeRegistration:
  criSocket: {{.CriSocket}}`)

const InitTemplateTextV1bate2 = string(`
apiVersion: kubeadm.k8s.io/v1beta2
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
  - 127.0.0.1
  - {{.ApiServer}}
  {{range .Masters -}}
  - {{.}}
  {{end -}}
  {{range .CertSANS -}}
  - {{.}}
  {{end -}}
  - {{.VIP}}
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
      listen-metrics-urls: http://0.0.0.0:2381`)
