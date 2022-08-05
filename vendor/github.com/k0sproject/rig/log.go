package rig

import "github.com/k0sproject/rig/log"

// SetLogger can be used to assign your own logger to rig
func SetLogger(logger log.Logger) {
	log.Log = logger
}

func init() {
	SetLogger(&log.StdLog{})
}
