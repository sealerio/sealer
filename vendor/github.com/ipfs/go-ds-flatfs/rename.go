//go:build !plan9
// +build !plan9

package flatfs

import "os"

var rename = os.Rename
