package runtime

//Metadata use file Metadata in rootfs to help cluster install.
type Metadata struct {
	Version string `json:"version"`
	Arch    string `json:"arch"`
	Variant string `json:"variant"`
	//KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion"`
	NydusFlag   bool   `json:"NydusFlag"`
	//ClusterRuntime is a flag to distinguish the runtime for k0s、k8s、k3s
	ClusterRuntime ClusterRuntime `json:"ClusterRuntime"`
}

type ClusterRuntime string

const (
	K0s ClusterRuntime = "k0s"
	K3s ClusterRuntime = "k3s"
	K8s ClusterRuntime = "k8s"
)
