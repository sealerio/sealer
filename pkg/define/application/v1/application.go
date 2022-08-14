package v1

import "github.com/sealerio/sealer/pkg/define/application/version"

type Application interface {
	version.VersionedApplication

	// helm, yaml, rawcmd
	Type() string

	UniqueName() string

	// 1. helm-specified cmd
	// 2. kube-specified cmd, like apply, create.
	// 3. rawcmd
	Cmd() string

	// not sure if this func is needed
	Configurations() string

	Info() string
}

// input application metadata, which will be generated from build
func NewV1Application() version.VersionedApplication {
	return nil
}
