package selector

import (
	"fmt"
	"io"
	"math"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
)

// Matcher marks a node to be included in the "result" set.
// (All nodes traversed by a selector are in the "covered" set (which is a.k.a.
// "the merkle proof"); the "result" set is a subset of the "covered" set.)
//
// In libraries using selectors, the "result" set is typically provided to
// some user-specified callback.
//
// A selector tree with only "explore*"-type selectors and no Matcher selectors
// is valid; it will just generate a "covered" set of nodes and no "result" set.
// TODO: From spec: implement conditions and labels
type Matcher struct {
	*Slice
}

// Slice limits a result node to a subset of the node.
// The returned node will be limited based on slicing the specified range of the
// node into a new node, or making use of the `AsLargeBytes` io.ReadSeeker to
// restrict response with a SectionReader.
//
// Slice supports [From,To) ranges, where From is inclusive and To is exclusive.
// Negative values for From and To are interpreted as offsets from the end of
// the node. If To is greater than the node length, it will be truncated to the
// node length. If From is greater than the node length or greater than To, the
// result will be a non-match.
type Slice struct {
	From int64
	To   int64
}

func sliceBounds(from, to, length int64) (bool, int64, int64) {
	if to < 0 {
		to = length + to
	} else if length < to {
		to = length
	}
	if from < 0 {
		from = length + from
		if from < 0 {
			from = 0
		}
	}
	if from > to || from >= length {
		return false, 0, 0
	}
	return true, from, to
}

func (s Slice) Slice(n datamodel.Node) (datamodel.Node, error) {
	var from, to int64
	switch n.Kind() {
	case datamodel.Kind_String:
		str, err := n.AsString()
		if err != nil {
			return nil, err
		}

		var match bool
		match, from, to = sliceBounds(s.From, s.To, int64(len(str)))
		if !match {
			return nil, nil
		}
		return basicnode.NewString(str[from:to]), nil
	case datamodel.Kind_Bytes:
		to = s.To
		from = s.From
		var length int64 = math.MaxInt64
		var rdr io.ReadSeeker
		var bytes []byte
		var err error

		if lbn, ok := n.(datamodel.LargeBytesNode); ok {
			rdr, err = lbn.AsLargeBytes()
			if err != nil {
				return nil, err
			}
			// calculate length from seeker
			length, err = rdr.Seek(0, io.SeekEnd)
			if err != nil {
				return nil, err
			}
			// reset
			_, err = rdr.Seek(0, io.SeekStart)
			if err != nil {
				return nil, err
			}
		} else {
			bytes, err = n.AsBytes()
			if err != nil {
				return nil, err
			}
			length = int64(len(bytes))
		}

		var match bool
		match, from, to = sliceBounds(from, to, length)
		if !match {
			return nil, nil
		}
		if rdr != nil {
			sr := io.NewSectionReader(&readerat{rdr, 0}, from, to-from)
			return basicnode.NewBytesFromReader(sr), nil
		}
		return basicnode.NewBytes(bytes[from:to]), nil
	default:
		return nil, nil
	}
}

// Interests are empty for a matcher (for now) because
// It is always just there to match, not explore further
func (s Matcher) Interests() []datamodel.PathSegment {
	return []datamodel.PathSegment{}
}

// Explore will return nil because a matcher is a terminal selector
func (s Matcher) Explore(n datamodel.Node, p datamodel.PathSegment) (Selector, error) {
	return nil, nil
}

// Decide is always true for a match cause it's in the result set
// Deprecated: use Match instead
func (s Matcher) Decide(n datamodel.Node) bool {
	return true
}

// Match is always true for a match cause it's in the result set
func (s Matcher) Match(node datamodel.Node) (datamodel.Node, error) {
	if s.Slice != nil {
		return s.Slice.Slice(node)
	}
	return node, nil
}

// ParseMatcher assembles a Selector
// from a matcher selector node
// TODO: Parse labels and conditions
func (pc ParseContext) ParseMatcher(n datamodel.Node) (Selector, error) {
	if n.Kind() != datamodel.Kind_Map {
		return nil, fmt.Errorf("selector spec parse rejected: selector body must be a map")
	}

	// check if a slice is specified
	if subset, err := n.LookupByString("subset"); err == nil {
		if subset.Kind() != datamodel.Kind_Map {
			return nil, fmt.Errorf("selector spec parse rejected: subset body must be a map")
		}
		from, err := subset.LookupByString("[")
		if err != nil {
			return nil, fmt.Errorf("selector spec parse rejected: selector body must be a map with a from '[' key")
		}
		fromN, err := from.AsInt()
		if err != nil {
			return nil, fmt.Errorf("selector spec parse rejected: selector body must be a map with a 'from' key that is a number")
		}
		to, err := subset.LookupByString("]")
		if err != nil {
			return nil, fmt.Errorf("selector spec parse rejected: selector body must be a map with a to ']' key")
		}
		toN, err := to.AsInt()
		if err != nil {
			return nil, fmt.Errorf("selector spec parse rejected: selector body must be a map with a 'to' key that is a number")
		}
		if toN >= 0 && fromN > toN {
			return nil, fmt.Errorf("selector spec parse rejected: selector body must be a map with a 'from' key that is less than or equal to the 'to' key")
		}
		return Matcher{&Slice{
			From: fromN,
			To:   toN,
		}}, nil
	}
	return Matcher{}, nil
}
