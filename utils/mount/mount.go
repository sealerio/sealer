// +build darwin

package mount

type Interface interface {
	// return mount target merged dir, if target is "", using default dir name : [dir hash]/merged
	Mount(target string, upperDir string, layers ...string) error
	Unmount(target string) error
}

func NewMountDriver() Interface {
	return &Default{}
}
