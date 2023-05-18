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

package kubeadm

const (
	InitConfiguration      = "InitConfiguration"
	JoinConfiguration      = "JoinConfiguration"
	ClusterConfiguration   = "ClusterConfiguration"
	KubeProxyConfiguration = "KubeProxyConfiguration"
	KubeletConfiguration   = "KubeletConfiguration"
)

const (
	DefaultKubeadmConfig = `
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
localAPIEndpoint:
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock

---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.19.8
imageRepository: sea.hub:5000
networking:
  podSubnet: 100.64.0.0/10
  serviceSubnet: 10.96.0.0/22
apiServer:
  extraArgs:
    feature-gates: TTLAfterFinished=true,EphemeralContainers=true
    audit-policy-file: "/etc/kubernetes/audit-policy.yml"
    audit-log-path: "/var/log/kubernetes/audit.log"
    audit-log-format: json
    audit-log-maxbackup: '10'
    audit-log-maxsize: '100'
    audit-log-maxage: '7'
    enable-aggregator-routing: 'true'
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

---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  excludeCIDRs:
    - "10.103.97.2/32"

---
apiVersion: kubelet.config.k8s.io/v1beta1
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
cgroupDriver:
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
serializeImagePulls: false
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 4h0m0s
syncFrequency: 1m0s
volumeStatsAggPeriod: 1m0s
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
caCertPath: /etc/kubernetes/pki/ca.crt
discovery:
  timeout: 5m0s
nodeRegistration:
  criSocket: /var/run/dockershim.sock
controlPlane:
  localAPIEndpoint:
    bindPort: 6443`
)

// RemovedFeatureGates
// key: featureGate
// value[0]: min/supported version
// value[1]: max/removed version
// https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates-removed/
var RemovedFeatureGates = map[string][2]string{
	"Accelerators":                      {"1.6", "1.11"},
	"AffinityInAnnotations":             {"1.6", "1.8"},
	"AllowExtTrafficLocalEndpoints":     {"1.4", "1.9"},
	"AllowInsecureBackendProxy":         {"1.17", "1.25"},
	"AttachVolumeLimit":                 {"1.11", "1.21"},
	"BalanceAttachedNodeVolumes":        {"1.11", "1.22"},
	"BlockVolume":                       {"1.9", "1.21"},
	"BoundServiceAccountTokenVolume":    {"1.13", "1.23"},
	"CRIContainerLogRotation":           {"1.10", "1.22"},
	"CSIBlockVolume":                    {"1.11", "1.21"},
	"CSIDriverRegistry":                 {"1.12", "1.21"},
	"CSIInlineVolume":                   {"1.15", "1.26"},
	"CSIMigration":                      {"1.14", "1.26"},
	"CSIMigrationAWS":                   {"1.14", "1.26"},
	"CSIMigrationAWSComplete":           {"1.17", "1.21"},
	"CSIMigrationAzureDisk":             {"1.15", "1.26"},
	"CSIMigrationAzureDiskComplete":     {"1.17", "1.21"},
	"CSIMigrationAzureFileComplete":     {"1.17", "1.21"},
	"CSIMigrationGCEComplete":           {"1.17", "1.21"},
	"CSIMigrationOpenStack":             {"1.14", "1.25"},
	"CSIMigrationOpenStackComplete":     {"1.17", "1.21"},
	"CSIMigrationvSphereComplete":       {"1.19", "1.22"},
	"CSINodeInfo":                       {"1.12", "1.22"},
	"CSIPersistentVolume":               {"1.9", "1.16"},
	"CSIServiceAccountToken":            {"1.20", "1.24"},
	"CSIVolumeFSGroupPolicy":            {"1.19", "1.25"},
	"CSRDuration":                       {"1.22", "1.25"},
	"ConfigurableFSGroupPolicy":         {"1.18", "1.25"},
	"ControllerManagerLeaderMigration":  {"1.21", "1.26"},
	"CronJobControllerV2":               {"1.20", "1.23"},
	"CustomPodDNS":                      {"1.9", "1.16"},
	"CustomResourceDefaulting":          {"1.15", "1.18"},
	"CustomResourcePublishOpenAPI":      {"1.14", "1.18"},
	"CustomResourceSubresources":        {"1.10", "1.18"},
	"CustomResourceValidation":          {"1.8", "1.18"},
	"CustomResourceWebhookConversion":   {"1.13", "1.18"},
	"DaemonSetUpdateSurge":              {"1.21", "1.26"},
	"DefaultPodTopologySpread":          {"1.19", "1.25"},
	"DynamicAuditing":                   {"1.13", "1.19"},
	"DynamicKubeletConfig":              {"1.4", "1.25"},
	"DynamicProvisioningScheduling":     {"1.11", "1.12"},
	"DynamicVolumeProvisioning":         {"1.3", "1.12"},
	"EnableAggregatedDiscoveryTimeout":  {"1.16", "1.17"},
	"EnableEquivalenceClassCache":       {"1.8", "1.23"},
	"EndpointSlice":                     {"1.16", "1.24"},
	"EndpointSliceNodeName":             {"1.20", "1.24"},
	"EndpointSliceProxying":             {"1.18", "1.24"},
	"EphemeralContainers":               {"1.16", "1.26"},
	"EvenPodsSpread":                    {"1.16", "1.21"},
	"ExpandCSIVolumes":                  {"1.14", "1.26"},
	"ExpandInUsePersistentVolumes":      {"1.11", "1.26"},
	"ExpandPersistentVolumes":           {"1.8", "1.26"},
	"ExperimentalCriticalPodAnnotation": {"1.5", "1.16"},
	"ExternalPolicyForExternalIP":       {"1.18", "1.22"},
	"GCERegionalPersistentDisk":         {"1.10", "1.16"},
	"GenericEphemeralVolume":            {"1.19", "1.24"},
	"HugePageStorageMediumSize":         {"1.18", "1.24"},
	"HugePages":                         {"1.8", "1.16"},
	"HyperVContainer":                   {"1.10", "1.20"},
	"IPv6DualStack":                     {"1.15", "1.24"},
	"IdentifyPodOS":                     {"1.23", "1.26"},
	"ImmutableEphemeralVolumes":         {"1.18", "1.24"},
	"IndexedJob":                        {"1.21", "1.25"},
	"IngressClassNamespacedParams":      {"1.21", "1.24"},
	"Initializers":                      {"1.7", "1.14"},
	"KubeletConfigFile":                 {"1.8", "1.10"},
	"KubeletPluginsWatcher":             {"1.11", "1.16"},
	"LegacyNodeRoleBehavior":            {"1.16", "1.22"},
	"LocalStorageCapacityIsolation":     {"1.7", "1.26"},
	"MountContainers":                   {"1.9", "1.17"},
	"MountPropagation":                  {"1.8", "1.14"},
	"NamespaceDefaultLabelName":         {"1.21", "1.23"},
	"NetworkPolicyEndPort":              {"1.21", "1.26"},
	"NodeDisruptionExclusion":           {"1.16", "1.22"},
	"NodeLease":                         {"1.12", "1.23"},
	"NonPreemptingPriority":             {"1.15", "1.25"},
	"PVCProtection":                     {"1.9", "1.10"},
	"PersistentLocalVolumes":            {"1.7", "1.16"},
	"PodAffinityNamespaceSelector":      {"1.21", "1.25"},
	"PodDisruptionBudget":               {"1.3", "1.25"},
	"PodOverhead":                       {"1.16", "1.25"},
	"PodPriority":                       {"1.8", "1.18"},
	"PodReadinessGates":                 {"1.11", "1.16"},
	"PodShareProcessNamespace":          {"1.10", "1.19"},
	"PreferNominatedNode":               {"1.21", "1.25"},
	"RequestManagement":                 {"1.15", "1.17"},
	"ResourceLimitsPriorityFunction":    {"1.9", "1.19"},
	"ResourceQuotaScopeSelectors":       {"1.11", "1.18"},
	"RootCAConfigMap":                   {"1.13", "1.22"},
	"RotateKubeletClientCertificate":    {"1.8", "1.21"},
	"RunAsGroup":                        {"1.14", "1.22"},
	"RuntimeClass":                      {"1.12", "1.24"},
	"SCTPSupport":                       {"1.12", "1.22"},
	"ScheduleDaemonSetPods":             {"1.11", "1.18"},
	"SelectorIndex":                     {"1.18", "1.25"},
	"ServiceAccountIssuerDiscovery":     {"1.18", "1.23"},
	"ServiceAppProtocol":                {"1.18", "1.22"},
	"ServiceLBNodePortControl":          {"1.20", "1.25"},
	"ServiceLoadBalancerClass":          {"1.21", "1.25"},
	"ServiceLoadBalancerFinalizer":      {"1.15", "1.20"},
	"ServiceNodeExclusion":              {"1.8", "1.22"},
	"ServiceTopology":                   {"1.17", "1.22"},
	"SetHostnameAsFQDN":                 {"1.19", "1.21"},
	"StartupProbe":                      {"1.16", "1.23"},
	"StatefulSetMinReadySeconds":        {"1.22", "1.26"},
	"StorageObjectInUseProtection":      {"1.10", "1.24"},
	"StreamingProxyRedirects":           {"1.5", "1.24"},
	"SupportIPVSProxyMode":              {"1.8", "1.20"},
	"SupportNodePidsLimit":              {"1.14", "1.23"},
	"SupportPodPidsLimit":               {"1.10", "1.23"},
	"SuspendJob":                        {"1.21", "1.25"},
	"Sysctls":                           {"1.11", "1.22"},
	"TTLAfterFinished":                  {"1.12", "1.24"},
	"TaintBasedEvictions":               {"1.6", "1.20"},
	"TaintNodesByCondition":             {"1.8", "1.18"},
	"TokenRequest":                      {"1.10", "1.21"},
	"TokenRequestProjection":            {"1.11", "1.21"},
	"ValidateProxyRedirects":            {"1.12", "1.24"},
	"VolumePVCDataSource":               {"1.15", "1.21"},
	"VolumeScheduling":                  {"1.9", "1.16"},
	"VolumeSnapshotDataSource":          {"1.12", "1.22"},
	"VolumeSubpath":                     {"1.10", "1.24"},
	"VolumeSubpathEnvExpansion":         {"1.14", "1.24"},
	"WarningHeaders":                    {"1.19", "1.24"},
	"WindowsEndpointSliceProxying":      {"1.19", "1.24"},
	"WindowsGMSA":                       {"1.14", "1.20"},
	"WindowsRunAsUserName":              {"1.16", "1.20"},
}
