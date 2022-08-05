package registry

import (
	"fmt"

	"github.com/k0sproject/rig"
)

type buildFunc = func() interface{}
type matchFunc = func(rig.OSVersion) bool

type osFactory struct {
	MatchFunc matchFunc
	BuildFunc buildFunc
}

var osModules []*osFactory

// RegisterOSModule registers a OS support module into rig's registry
func RegisterOSModule(mf matchFunc, bf buildFunc) {
	// Inserting to beginning to match the most latest added
	osModules = append([]*osFactory{{MatchFunc: mf, BuildFunc: bf}}, osModules...)
}

// GetOSModuleBuilder returns a suitable OS support module from rig's registry
func GetOSModuleBuilder(osv rig.OSVersion) (buildFunc, error) {
	for _, of := range osModules {
		if of.MatchFunc(osv) {
			return of.BuildFunc, nil
		}
	}

	return nil, fmt.Errorf("os support module not found")
}
