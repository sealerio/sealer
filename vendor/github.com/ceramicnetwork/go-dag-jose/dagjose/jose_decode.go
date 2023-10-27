package dagjose

import (
	"bytes"
	"io"

	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
)

// DecodeOptions can be used to customize the behavior of a decoding function. The Decode method on this struct fits the
// codec.Decoder function interface.
type DecodeOptions struct {
	// If true and the `payload` field is present, add a `link` field corresponding to the `payload`.
	AddLink bool
}

// Decode deserializes data from the given io.Reader and feeds it into the given datamodel.NodeAssembler. Decode fits
// the codec.Decoder function interface.
func (cfg DecodeOptions) Decode(na datamodel.NodeAssembler, r io.Reader) error {
	// Since a JWE might be partially decoded by the time we figure out that this isn't a JWE and is probably a JWS, we
	// can't pass the same reader to both decode methods. Instead, we read off all input bytes and use them in separate
	// readers to the JWE/JWS decode methods.
	if buf, err := io.ReadAll(r); err != nil {
		return err
	} else if err := cfg.DecodeJWE(na, bytes.NewReader(buf)); err != nil {
		return cfg.DecodeJWS(na, bytes.NewReader(buf))
	}
	return nil
}

// Decode deserializes data from the given io.Reader and feeds it into the given datamodel.NodeAssembler. Decode fits
// the codec.Decoder function interface.
func Decode(na datamodel.NodeAssembler, r io.Reader) error {
	return DecodeOptions{
		AddLink: true,
	}.Decode(na, r)
}

func (DecodeOptions) DecodeJWE(na datamodel.NodeAssembler, r io.Reader) error {
	// Check for the fastpath where the passed assembler is already of type `_DecodedJWE__ReprBuilder` or
	// `_DecodedJWE__ReprAssembler`.
	copyRequired := false
	jweBuilder, castOk := na.(*_DecodedJWE__ReprBuilder)
	if !castOk {
		// This could still be `_DecodedJWE__ReprAssembler`, so check for that.
		_, castOk := na.(*_DecodedJWE__ReprAssembler)
		if !castOk {
			// No fastpath possible, just create a new `_DecodedJWE__ReprBuilder`, use it, then copy the built node into
			// the assembler the caller passed in.
			jweBuilder = Type.DecodedJWE__Repr.NewBuilder().(*_DecodedJWE__ReprBuilder)
			copyRequired = true
		}
	}
	// DAG-CBOR is a superset of DAG-JOSE and can be used to decode valid DAG-JOSE objects.
	// See: https://specs.ipld.io/block-layer/codecs/dag-jose.html
	if err := dagcbor.Decode(jweBuilder, r); err != nil {
		return err
	}
	// The "representation" node gives an accurate view of fields that are actually present
	jweNode := jweBuilder.Build().(schema.TypedNode).Representation()
	if copyRequired {
		return datamodel.Copy(jweNode, na)
	}
	return nil
}

func (cfg DecodeOptions) DecodeJWS(na datamodel.NodeAssembler, r io.Reader) error {
	// Check for the fastpath where the passed assembler is already of type `_DecodedJWS__ReprBuilder` or
	// `_DecodedJWS__ReprAssembler`.
	copyRequired := false
	jwsBuilder, castOk := na.(*_DecodedJWS__ReprBuilder)
	if !castOk {
		// This could still be `_DecodedJWS__ReprAssembler`, so check for that.
		_, castOk := na.(*_DecodedJWS__ReprAssembler)
		if !castOk {
			// No fastpath possible, just create a new `_DecodedJWE__ReprBuilder`, use it, then copy the built node into
			// the assembler the caller passed in.
			jwsBuilder = Type.DecodedJWS__Repr.NewBuilder().(*_DecodedJWS__ReprBuilder)
			copyRequired = true
		}
	}
	// DAG-CBOR is a superset of DAG-JOSE and can be used to decode valid DAG-JOSE objects.
	// See: https://specs.ipld.io/block-layer/codecs/dag-jose.html
	if err := dagcbor.Decode(jwsBuilder, r); err != nil {
		return err
	}
	if cfg.AddLink {
		// If `payload` is present but `link` is not, add `link` with the corresponding encoded CID.
		linkNode := &jwsBuilder.w.link
		if !linkNode.Exists() {
			if link, err := Type.Base64Url.Link(&jwsBuilder.w.payload); err != nil {
				return err
			} else {
				linkNode.m = schema.Maybe_Value
				linkNode.v = *link
			}
		}
	}
	// The "representation" node gives an accurate view of fields that are actually present
	jwsNode := jwsBuilder.Build().(schema.TypedNode).Representation()
	if copyRequired {
		return datamodel.Copy(jwsNode, na)
	}
	return nil
}
