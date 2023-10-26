package dagjose

//go:generate go run ./gen .
//go:generate go fmt ./

import (
	"github.com/ipld/go-ipld-prime/multicodec"
)

func init() {
	multicodec.RegisterDecoder(0x85, Decode)
	multicodec.RegisterEncoder(0x85, Encode)
}
