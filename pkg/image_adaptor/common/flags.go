package common

type BuildFlags struct {
	BuildType string
	Kubefile  string
	Tags      []string
	NoCache   bool
	Base      bool
	BuildArgs []string
	Platform  string
}
