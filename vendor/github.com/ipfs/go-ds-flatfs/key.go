package flatfs

import (
	"github.com/ipfs/go-datastore"
)

// keyIsValid returns true if the key is valid for flatfs.
// Allows keys that match [0-9A-Z+-_=].
func keyIsValid(key datastore.Key) bool {
	ks := key.String()
	if len(ks) < 2 || ks[0] != '/' {
		return false
	}
	for _, b := range ks[1:] {
		if '0' <= b && b <= '9' {
			continue
		}
		if 'A' <= b && b <= 'Z' {
			continue
		}
		switch b {
		case '+', '-', '_', '=':
			continue
		}
		return false
	}
	return true
}
