package rig

// OSVersion host operating system version information
type OSVersion struct {
	ID      string
	IDLike  string
	Name    string
	Version string
}

// String returns a human readable representation of OSVersion
func (o *OSVersion) String() string {
	if o.Name != "" {
		return o.Name
	}
	return o.ID + " " + o.Version
}
