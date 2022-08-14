package os

import (
	"github.com/k0sproject/rig/exec"
)

// Host is an interface to a host object that has the functions needed by the various OS support packages
type Host interface {
	Upload(source, destination string, opts ...exec.Option) error
	Exec(string, ...exec.Option) error
	ExecOutput(string, ...exec.Option) (string, error)
	Execf(string, ...interface{}) error
	ExecOutputf(string, ...interface{}) (string, error)
	String() string
	Sudo(string) (string, error)
}
