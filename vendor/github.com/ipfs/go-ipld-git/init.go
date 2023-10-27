package ipldgit

//go:generate go run ./gen .
//go:generate go fmt ./

import (
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	mc "github.com/ipld/go-ipld-prime/multicodec"
)

var (
	_ ipld.Decoder = Decode
	_ ipld.Encoder = Encode
)

func init() {
	mc.RegisterEncoder(cid.GitRaw, Encode)
	mc.RegisterDecoder(cid.GitRaw, Decode)
}
