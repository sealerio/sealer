package dagcbor

import (
	"io"

	"github.com/ipld/go-ipld-prime/codec"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/multicodec"
)

var (
	_ codec.Decoder = Decode
	_ codec.Encoder = Encode
)

func init() {
	multicodec.RegisterEncoder(0x71, Encode)
	multicodec.RegisterDecoder(0x71, Decode)
}

// Decode deserializes data from the given io.Reader and feeds it into the given datamodel.NodeAssembler.
// Decode fits the codec.Decoder function interface.
//
// A similar function is available on DecodeOptions type if you would like to customize any of the decoding details.
// This function uses the defaults for the dag-cbor codec
// (meaning: links (indicated by tag 42) are decoded).
//
// This is the function that will be registered in the default multicodec registry during package init time.
func Decode(na datamodel.NodeAssembler, r io.Reader) error {
	return DecodeOptions{
		AllowLinks: true,
	}.Decode(na, r)
}

// Encode walks the given datamodel.Node and serializes it to the given io.Writer.
// Encode fits the codec.Encoder function interface.
//
// A similar function is available on EncodeOptions type if you would like to customize any of the encoding details.
// This function uses the defaults for the dag-cbor codec
// (meaning: links are encoded, and map keys are sorted (with RFC7049 ordering!) during encode).
//
// This is the function that will be registered in the default multicodec registry during package init time.
func Encode(n datamodel.Node, w io.Writer) error {
	return EncodeOptions{
		AllowLinks:  true,
		MapSortMode: codec.MapSortMode_RFC7049,
	}.Encode(n, w)
}
