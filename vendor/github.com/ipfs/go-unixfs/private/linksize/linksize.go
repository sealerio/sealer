package linksize

import "github.com/ipfs/go-cid"

var LinkSizeFunction func(linkName string, linkCid cid.Cid) int
