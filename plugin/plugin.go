package plugin

import (
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type Interface interface {
	Run(context Context, phase Phase) error
}

type Phase string

const (
	PhasePreInit     = Phase("PreInit")
	PhasePreInstall  = Phase("PreInstall")
	PhasePostInstall = Phase("PostInstall")
)

type Context struct {
	Cluster *v1.Cluster
	Plugin  *v1.Plugin
}
