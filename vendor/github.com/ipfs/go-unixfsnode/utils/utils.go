package utils

import dagpb "github.com/ipld/go-codec-dagpb"

// Lookup finds a name key in a list of dag pb links
func Lookup(links dagpb.PBLinks, key string) dagpb.Link {
	li := links.Iterator()
	for !li.Done() {
		_, next := li.Next()
		name := ""
		if next.FieldName().Exists() {
			name = next.FieldName().Must().String()
		}
		if key == name {
			return next.FieldHash()
		}
	}
	return nil
}
