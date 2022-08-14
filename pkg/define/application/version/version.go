package version

// TODO maybe move it a global version interface, for version compatiblity
type VersionedApplication interface {
	Version() string
}
