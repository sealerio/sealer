package dagjose

import (
	"errors"
	"io"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/multiformats/go-multibase"
)

// Encode walks the given datamodel.Node and serializes it to the given io.Writer. Encode fits the codec.Encoder
// function interface.
func Encode(n datamodel.Node, w io.Writer) error {
	// "flattened" fields are not included in the schema and thus never encoded. That means that this cannot have been
	// called on a JOSE-related node because we wouldn't have gotten this far without an error have occurred earlier.
	// We'll assume this is some sort of Map-type node that we can reconstruct to be in a "general" form before any
	// actual JOSE-related operations are performed on it.
	if jwe, err := isJWE(n); err != nil {
		return err
	} else if jwe {
		if n, err := unflattenJWE(n); err != nil {
			return err
		} else if err := EncodeJWE(n, w); err != nil {
			return err
		}
	} else if jws, err := isJWS(n); err != nil {
		return err
	} else if jws {
		if n, err := unflattenJWS(n); err != nil {
			return err
		} else if err := EncodeJWS(n, w); err != nil {
			return err
		}
	} else {
		return errors.New("invalid JOSE object")
	}
	return nil
}

func EncodeJWE(n datamodel.Node, w io.Writer) error {
	// Check for the fastpath where the passed node is already of type `_EncodedJWE__Repr` or `_EncodedJWE`
	if _, castOk := n.(*_EncodedJWE__Repr); !castOk {
		// This could still be `_EncodedJWE`, so check for that.
		if _, castOk := n.(*_EncodedJWE); !castOk {
			// No fastpath possible, just create a new `_EncodedJWE__ReprBuilder` and copy the passed node into it.
			jweBuilder := Type.EncodedJWE__Repr.NewBuilder().(*_EncodedJWE__ReprBuilder)
			if err := datamodel.Copy(n, jweBuilder); err != nil {
				return err
			}
			n = jweBuilder.Build()
		}
		// The "representation" node gives an accurate view of fields that are actually present
		n = n.(schema.TypedNode).Representation()
	}
	// DAG-CBOR is a superset of DAG-JOSE and can be used to encode valid DAG-JOSE objects.
	// See: https://specs.ipld.io/block-layer/codecs/dag-jose.html
	return dagcbor.EncodeOptions{
		MapSortMode: codec.MapSortMode_RFC7049,
	}.Encode(n, w)
}

func EncodeJWS(n datamodel.Node, w io.Writer) error {
	// Check for the fastpath where the passed node is already of type `_EncodedJWES__Repr` or `_EncodedJWS`
	if _, castOk := n.(*_EncodedJWS__Repr); !castOk {
		// This could still be `_EncodedJWS`, so check for that.
		if _, castOk := n.(*_EncodedJWS); !castOk {
			// If `link` and `payload` are present, make sure they match.
			if err := validateLink(n); err != nil {
				return err
			}
			// No fastpath possible, just create a new `_EncodedJWS__ReprBuilder` and copy the passed node into it.
			jwsBuilder := Type.EncodedJWS__Repr.NewBuilder().(*_EncodedJWS__ReprBuilder)
			if err := datamodel.Copy(n, jwsBuilder); err != nil {
				return err
			}
			n = jwsBuilder.Build()
		}
		// The "representation" node gives an accurate view of fields that are actually present
		n = n.(schema.TypedNode).Representation()
	}
	// DAG-CBOR is a superset of DAG-JOSE and can be used to encode valid DAG-JOSE objects.
	// See: https://specs.ipld.io/block-layer/codecs/dag-jose.html
	return dagcbor.EncodeOptions{
		MapSortMode: codec.MapSortMode_RFC7049,
	}.Encode(n, w)
}

func validateLink(n datamodel.Node) error {
	if linkNode, err := lookupIgnoreNoSuchField("link", n); err != nil {
		return err
	} else if linkNode != nil {
		// If `link` was present then `payload` must be present and the two must match. If any error occurs here
		// (including `payload` being absent) return it.
		if payloadNode, err := n.LookupByString("payload"); err != nil {
			return err
		} else if payloadString, err := payloadNode.AsString(); err != nil {
			return err
		} else if cidFromPayload, err := cid.Decode(string(multibase.Base64url) + payloadString); err != nil {
			return err
		} else if linkFromNode, err := linkNode.AsLink(); err != nil {
			return err
		} else if linkFromNode.(cidlink.Link).Cid != cidFromPayload {
			return errors.New("cid mismatch")
		}
	}
	return nil
}
