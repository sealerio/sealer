package version

import "strings"

var (
	// Version of the product, is set during the build
	Version = "0.0.0"
	// GitCommit is set during the build
	GitCommit = "HEAD"
	// Environment of the product, is set during the build
	Environment = "development"
)

// IsPre is true when the current version is a prerelease
func IsPre() bool {
	return strings.Contains(Version, "-")
}
