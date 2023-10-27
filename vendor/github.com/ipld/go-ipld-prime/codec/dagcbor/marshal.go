package dagcbor

import (
	"fmt"
	"io"
	"sort"

	"github.com/polydawn/refmt/cbor"
	"github.com/polydawn/refmt/shared"
	"github.com/polydawn/refmt/tok"

	"github.com/ipld/go-ipld-prime/codec"
	"github.com/ipld/go-ipld-prime/datamodel"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// This file should be identical to the general feature in the parent package,
// except for the `case datamodel.Kind_Link` block,
// which is dag-cbor's special sauce for schemafree links.

// EncodeOptions can be used to customize the behavior of an encoding function.
// The Encode method on this struct fits the codec.Encoder function interface.
type EncodeOptions struct {
	// If true, allow encoding of Link nodes as CBOR tag(42);
	// otherwise, reject them as unencodable.
	AllowLinks bool

	// Control the sorting of map keys, using one of the `codec.MapSortMode_*` constants.
	MapSortMode codec.MapSortMode
}

// Encode walks the given datamodel.Node and serializes it to the given io.Writer.
// Encode fits the codec.Encoder function interface.
//
// The behavior of the encoder can be customized by setting fields in the EncodeOptions struct before calling this method.
func (cfg EncodeOptions) Encode(n datamodel.Node, w io.Writer) error {
	// Probe for a builtin fast path.  Shortcut to that if possible.
	type detectFastPath interface {
		EncodeDagCbor(io.Writer) error
	}
	if n2, ok := n.(detectFastPath); ok {
		return n2.EncodeDagCbor(w)
	}
	// Okay, generic inspection path.
	return Marshal(n, cbor.NewEncoder(w), cfg)
}

// Future work: we would like to remove the Marshal function,
// and in particular, stop seeing types from refmt (like shared.TokenSink) be visible.
// Right now, some kinds of configuration (e.g. for whitespace and prettyprint) are only available through interacting with the refmt types;
// we should improve our API so that this can be done with only our own types in this package.

// Marshal is a deprecated function.
// Please consider switching to EncodeOptions.Encode instead.
func Marshal(n datamodel.Node, sink shared.TokenSink, options EncodeOptions) error {
	var tk tok.Token
	return marshal(n, &tk, sink, options)
}

func marshal(n datamodel.Node, tk *tok.Token, sink shared.TokenSink, options EncodeOptions) error {
	switch n.Kind() {
	case datamodel.Kind_Invalid:
		return fmt.Errorf("cannot traverse a node that is absent")
	case datamodel.Kind_Null:
		tk.Type = tok.TNull
		_, err := sink.Step(tk)
		return err
	case datamodel.Kind_Map:
		return marshalMap(n, tk, sink, options)
	case datamodel.Kind_List:
		// Emit start of list.
		tk.Type = tok.TArrOpen
		l := n.Length()
		tk.Length = int(l) // TODO: overflow check
		if _, err := sink.Step(tk); err != nil {
			return err
		}
		// Emit list contents (and recurse).
		for i := int64(0); i < l; i++ {
			v, err := n.LookupByIndex(i)
			if err != nil {
				return err
			}
			if err := marshal(v, tk, sink, options); err != nil {
				return err
			}
		}
		// Emit list close.
		tk.Type = tok.TArrClose
		_, err := sink.Step(tk)
		return err
	case datamodel.Kind_Bool:
		v, err := n.AsBool()
		if err != nil {
			return err
		}
		tk.Type = tok.TBool
		tk.Bool = v
		_, err = sink.Step(tk)
		return err
	case datamodel.Kind_Int:
		if uin, ok := n.(datamodel.UintNode); ok {
			v, err := uin.AsUint()
			if err != nil {
				return err
			}
			tk.Type = tok.TUint
			tk.Uint = v
		} else {
			v, err := n.AsInt()
			if err != nil {
				return err
			}
			tk.Type = tok.TInt
			tk.Int = v
		}
		_, err := sink.Step(tk)
		return err
	case datamodel.Kind_Float:
		v, err := n.AsFloat()
		if err != nil {
			return err
		}
		tk.Type = tok.TFloat64
		tk.Float64 = v
		_, err = sink.Step(tk)
		return err
	case datamodel.Kind_String:
		v, err := n.AsString()
		if err != nil {
			return err
		}
		tk.Type = tok.TString
		tk.Str = v
		_, err = sink.Step(tk)
		return err
	case datamodel.Kind_Bytes:
		v, err := n.AsBytes()
		if err != nil {
			return err
		}
		tk.Type = tok.TBytes
		tk.Bytes = v
		_, err = sink.Step(tk)
		return err
	case datamodel.Kind_Link:
		if !options.AllowLinks {
			return fmt.Errorf("cannot Marshal ipld links to CBOR")
		}
		v, err := n.AsLink()
		if err != nil {
			return err
		}
		switch lnk := v.(type) {
		case cidlink.Link:
			if !lnk.Cid.Defined() {
				return fmt.Errorf("encoding undefined CIDs are not supported by this codec")
			}
			tk.Type = tok.TBytes
			tk.Bytes = append([]byte{0}, lnk.Bytes()...)
			tk.Tagged = true
			tk.Tag = linkTag
			_, err = sink.Step(tk)
			tk.Tagged = false
			return err
		default:
			return fmt.Errorf("schemafree link emission only supported by this codec for CID type links")
		}
	default:
		panic("unreachable")
	}
}

func marshalMap(n datamodel.Node, tk *tok.Token, sink shared.TokenSink, options EncodeOptions) error {
	// Emit start of map.
	tk.Type = tok.TMapOpen
	expectedLength := int(n.Length())
	tk.Length = expectedLength // TODO: overflow check
	if _, err := sink.Step(tk); err != nil {
		return err
	}
	if options.MapSortMode != codec.MapSortMode_None {
		// Collect map entries, then sort by key
		type entry struct {
			key   string
			value datamodel.Node
		}
		entries := []entry{}
		for itr := n.MapIterator(); !itr.Done(); {
			k, v, err := itr.Next()
			if err != nil {
				return err
			}
			keyStr, err := k.AsString()
			if err != nil {
				return err
			}
			entries = append(entries, entry{keyStr, v})
		}
		if len(entries) != expectedLength {
			return fmt.Errorf("map Length() does not match number of MapIterator() entries")
		}
		// Apply the desired sort function.
		switch options.MapSortMode {
		case codec.MapSortMode_Lexical:
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].key < entries[j].key
			})
		case codec.MapSortMode_RFC7049:
			sort.Slice(entries, func(i, j int) bool {
				// RFC7049 style sort as per DAG-CBOR spec
				li, lj := len(entries[i].key), len(entries[j].key)
				if li == lj {
					return entries[i].key < entries[j].key
				}
				return li < lj
			})
		}
		// Emit map contents (and recurse).
		for _, e := range entries {
			tk.Type = tok.TString
			tk.Str = e.key
			if _, err := sink.Step(tk); err != nil {
				return err
			}
			if err := marshal(e.value, tk, sink, options); err != nil {
				return err
			}
		}
	} else { // no sorting
		// Emit map contents (and recurse).
		var entryCount int
		for itr := n.MapIterator(); !itr.Done(); {
			k, v, err := itr.Next()
			if err != nil {
				return err
			}
			entryCount++
			tk.Type = tok.TString
			tk.Str, err = k.AsString()
			if err != nil {
				return err
			}
			if _, err := sink.Step(tk); err != nil {
				return err
			}
			if err := marshal(v, tk, sink, options); err != nil {
				return err
			}
		}
		if entryCount != expectedLength {
			return fmt.Errorf("map Length() does not match number of MapIterator() entries")
		}
	}
	// Emit map close.
	tk.Type = tok.TMapClose
	_, err := sink.Step(tk)
	return err
}

// EncodedLength will calculate the length in bytes that the encoded form of the
// provided Node will occupy.
//
// Note that this function requires a full walk of the Node's graph, which may
// not necessarily be a trivial cost and will incur some allocations. Using this
// method to calculate buffers to pre-allocate may not result in performance
// gains, but rather incur an overall cost. Use with care.
func EncodedLength(n datamodel.Node) (int64, error) {
	switch n.Kind() {
	case datamodel.Kind_Invalid:
		return 0, fmt.Errorf("cannot traverse a node that is absent")
	case datamodel.Kind_Null:
		return 1, nil // 0xf6
	case datamodel.Kind_Map:
		length := uintLength(uint64(n.Length())) // length prefixed major 5
		for itr := n.MapIterator(); !itr.Done(); {
			k, v, err := itr.Next()
			if err != nil {
				return 0, err
			}
			keyLength, err := EncodedLength(k)
			if err != nil {
				return 0, err
			}
			length += keyLength
			valueLength, err := EncodedLength(v)
			if err != nil {
				return 0, err
			}
			length += valueLength
		}
		return length, nil
	case datamodel.Kind_List:
		nl := n.Length()
		length := uintLength(uint64(nl)) // length prefixed major 4
		for i := int64(0); i < nl; i++ {
			v, err := n.LookupByIndex(i)
			if err != nil {
				return 0, err
			}
			innerLength, err := EncodedLength(v)
			if err != nil {
				return 0, err
			}
			length += innerLength
		}
		return length, nil
	case datamodel.Kind_Bool:
		return 1, nil // 0xf4 or 0xf5
	case datamodel.Kind_Int:
		v, err := n.AsInt()
		if err != nil {
			return 0, err
		}
		if v < 0 {
			v = -v - 1 // negint is stored as one less than actual
		}
		return uintLength(uint64(v)), nil // major 0 or 1, as small as possible
	case datamodel.Kind_Float:
		return 9, nil // always major 7 and 64-bit float
	case datamodel.Kind_String:
		v, err := n.AsString()
		if err != nil {
			return 0, err
		}

		return uintLength(uint64(len(v))) + int64(len(v)), nil // length prefixed major 3
	case datamodel.Kind_Bytes:
		v, err := n.AsBytes()
		if err != nil {
			return 0, err
		}
		return uintLength(uint64(len(v))) + int64(len(v)), nil // length prefixed major 2
	case datamodel.Kind_Link:
		v, err := n.AsLink()
		if err != nil {
			return 0, err
		}
		switch lnk := v.(type) {
		case cidlink.Link:
			length := int64(2)                    // tag,42: 0xd82a
			bl := int64(len(lnk.Bytes())) + 1     // additional 0x00 in front of the CID bytes
			length += uintLength(uint64(bl)) + bl // length prefixed major 2
			return length, err
		default:
			return 0, fmt.Errorf("schemafree link emission only supported by this codec for CID type links")
		}
	default:
		panic("unreachable")
	}
}

// Calculate how many bytes an integer, and therefore also the leading bytes of
// a length-prefixed token. CBOR will pack it up into the smallest possible
// uint representation, even merging it with the major if it's <=23.

type boundaryLength struct {
	upperBound uint64
	length     int64
}

var lengthBoundaries = []boundaryLength{
	{24, 1},         // packed major|minor
	{256, 2},        // major, 8-bit length
	{65536, 3},      // major, 16-bit length
	{4294967296, 5}, // major, 32-bit length
	{0, 9},          // major, 64-bit length
}

func uintLength(ii uint64) int64 {
	for _, lb := range lengthBoundaries {
		if ii < lb.upperBound {
			return lb.length
		}
	}
	// maximum number of bytes to pack this int
	// if this int is used as a length prefix for a map, list, string or bytes
	// then we likely have a very bad Node that shouldn't be encoded, but the
	// encoder may raise problems with that if the memory allocator doesn't first.
	return lengthBoundaries[len(lengthBoundaries)-1].length
}
