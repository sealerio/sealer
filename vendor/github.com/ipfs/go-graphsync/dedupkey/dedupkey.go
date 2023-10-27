package dedupkey

import (
	basicnode "github.com/ipld/go-ipld-prime/node/basic"

	"github.com/ipfs/go-graphsync/ipldutil"
)

// EncodeDedupKey returns encoded cbor data for string key
func EncodeDedupKey(key string) ([]byte, error) {
	nb := basicnode.Prototype.String.NewBuilder()
	err := nb.AssignString(key)
	if err != nil {
		return nil, err
	}
	nd := nb.Build()
	return ipldutil.EncodeNode(nd)
}

// DecodeDedupKey returns a string key decoded from cbor data
func DecodeDedupKey(data []byte) (string, error) {
	nd, err := ipldutil.DecodeNode(data)
	if err != nil {
		return "", err
	}
	return nd.AsString()
}
