package installer

import (
	"github.com/sealerio/sealer/pkg/define/application/version"
)

type Installer struct {
	// Not sure if the app installer needs to care about infra.
	// So ensureRootfsMounted is the minimum demand for app installer.
	ensureRootfsMounted func(image string) (bool, error)
	executors           map[string]executor
	// app installer needs a cluster client interface, which is used to do the following operations to cluster:
	// 1. "kubectl apply -f"
	// 2. "helm ..."
	// 3. ......
	//clusterClient
}

func (installer *Installer) AppendApps(apps []version.VersionedApplication) error {
	return installer.registerExecutors(apps)
}

func (installer *Installer) Exec() error {
	// do install operations here
	return nil
}

func (installer *Installer) registerExecutors(apps []version.VersionedApplication) error {
	return nil
}

func NewInstaller(ensureRootfsMounted func(image string) (bool, error), apps []version.VersionedApplication) (*Installer, error) {
	return &Installer{}, nil
}
