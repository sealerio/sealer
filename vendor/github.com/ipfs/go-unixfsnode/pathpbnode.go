package unixfsnode

import (
	"github.com/ipfs/go-unixfsnode/iter"
	"github.com/ipfs/go-unixfsnode/utils"
	dagpb "github.com/ipld/go-codec-dagpb"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

var _ ipld.Node = PathedPBNode(nil)
var _ schema.TypedNode = PathedPBNode(nil)
var _ ipld.ADL = PathedPBNode(nil)

type PathedPBNode = *_PathedPBNode

type _PathedPBNode struct {
	_substrate dagpb.PBNode
}

func (n PathedPBNode) Kind() ipld.Kind {
	return n._substrate.Kind()
}

// LookupByString looks for the key in the list of links with a matching name
func (n PathedPBNode) LookupByString(key string) (ipld.Node, error) {
	links := n._substrate.FieldLinks()
	link := utils.Lookup(links, key)
	if link == nil {
		return nil, schema.ErrNoSuchField{Type: nil /*TODO*/, Field: ipld.PathSegmentOfString(key)}
	}
	return link, nil
}

func (n PathedPBNode) LookupByNode(key ipld.Node) (ipld.Node, error) {
	ks, err := key.AsString()
	if err != nil {
		return nil, err
	}
	return n.LookupByString(ks)
}

func (n PathedPBNode) LookupByIndex(idx int64) (ipld.Node, error) {
	return n._substrate.LookupByIndex(idx)
}

func (n PathedPBNode) LookupBySegment(seg ipld.PathSegment) (ipld.Node, error) {
	return n.LookupByString(seg.String())
}

func (n PathedPBNode) MapIterator() ipld.MapIterator {
	return iter.NewUnixFSDirMapIterator(n._substrate.Links.Iterator(), nil)
}

// ListIterator returns an iterator which yields key-value pairs
// traversing the node.
// If the node kind is anything other than a list, nil will be returned.
//
// The iterator will yield every entry in the list; that is, it
// can be expected that itr.Next will be called node.Length times
// before itr.Done becomes true.
func (n PathedPBNode) ListIterator() ipld.ListIterator {
	return nil
}

// Length returns the length of a list, or the number of entries in a map,
// or -1 if the node is not of list nor map kind.
func (n PathedPBNode) Length() int64 {
	return n._substrate.FieldLinks().Length()
}

func (n PathedPBNode) IsAbsent() bool {
	return false
}

func (n PathedPBNode) IsNull() bool {
	return false
}

func (n PathedPBNode) AsBool() (bool, error) {
	return n._substrate.AsBool()
}

func (n PathedPBNode) AsInt() (int64, error) {
	return n._substrate.AsInt()
}

func (n PathedPBNode) AsFloat() (float64, error) {
	return n._substrate.AsFloat()
}

func (n PathedPBNode) AsString() (string, error) {
	return n._substrate.AsString()
}

func (n PathedPBNode) AsBytes() ([]byte, error) {
	return n._substrate.AsBytes()
}

func (n PathedPBNode) AsLink() (ipld.Link, error) {
	return n._substrate.AsLink()
}

func (n PathedPBNode) Prototype() ipld.NodePrototype {
	// TODO: should this return something?
	// probobly not until we write the write interfaces
	return nil
}

// satisfy schema.TypedNode
func (PathedPBNode) Type() schema.Type {
	return nil /*TODO:typelit*/
}

func (n PathedPBNode) Representation() ipld.Node {
	return n._substrate.Representation()
}

// Native map accessors

func (n PathedPBNode) Iterator() *iter.UnixFSDir__Itr {

	return iter.NewUnixFSDirIterator(n._substrate.Links.Iterator(), nil)
}

func (n PathedPBNode) Lookup(key dagpb.String) dagpb.Link {
	return utils.Lookup(n._substrate.FieldLinks(), key.String())
}

// direct access to the links and data

func (n PathedPBNode) FieldLinks() dagpb.PBLinks {
	return n._substrate.FieldLinks()
}

func (n PathedPBNode) FieldData() dagpb.MaybeBytes {
	return n._substrate.FieldData()
}

// Substrate returns the underlying PBNode -- note: only the substrate will encode successfully to protobuf if writing
func (n PathedPBNode) Substrate() ipld.Node {
	return n._substrate
}
