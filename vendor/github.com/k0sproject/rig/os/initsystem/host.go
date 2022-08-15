package initsystem

// Host interface for init system
type Host interface {
	Execf(string, ...interface{}) error
	ExecOutputf(string, ...interface{}) (string, error)
	Sudo(string) (string, error)
}
