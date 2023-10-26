package directory

import (
	"context"

	"github.com/ipfs/go-unixfsnode/data"
	"github.com/ipfs/go-unixfsnode/iter"
	"github.com/ipfs/go-unixfsnode/utils"
	dagpb "github.com/ipld/go-codec-dagpb"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

var _ ipld.Node = UnixFSBasicDir(nil)
var _ schema.TypedNode = UnixFSBasicDir(nil)
var _ ipld.ADL = UnixFSBasicDir(nil)

type UnixFSBasicDir = *_UnixFSBasicDir

type _UnixFSBasicDir struct {
	_substrate dagpb.PBNode
}

func NewUnixFSBasicDir(ctx context.Context, substrate dagpb.PBNode, nddata data.UnixFSData, _ *ipld.LinkSystem) (ipld.Node, error) {
	if nddata.FieldDataType().Int() != data.Data_Directory {
		return nil, data.ErrWrongNodeType{Expected: data.Data_Directory, Actual: nddata.FieldDataType().Int()}
	}
	return &_UnixFSBasicDir{_substrate: substrate}, nil
}

func (n UnixFSBasicDir) Kind() ipld.Kind {
	return n._substrate.Kind()
}

// LookupByString looks for the key in the list of links with a matching name
func (n UnixFSBasicDir) LookupByString(key string) (ipld.Node, error) {
	links := n._substrate.FieldLinks()
	link := utils.Lookup(links, key)
	if link == nil {
		return nil, schema.ErrNoSuchField{Type: nil /*TODO*/, Field: ipld.PathSegmentOfString(key)}
	}
	return link, nil
}

func (n UnixFSBasicDir) LookupByNode(key ipld.Node) (ipld.Node, error) {
	ks, err := key.AsString()
	if err != nil {
		return nil, err
	}
	return n.LookupByString(ks)
}

func (n UnixFSBasicDir) LookupByIndex(idx int64) (ipld.Node, error) {
	return n._substrate.LookupByIndex(idx)
}

func (n UnixFSBasicDir) LookupBySegment(seg ipld.PathSegment) (ipld.Node, error) {
	return n.LookupByString(seg.String())
}

func (n UnixFSBasicDir) MapIterator() ipld.MapIterator {
	return iter.NewUnixFSDirMapIterator(n._substrate.Links.Iterator(), nil)
}

// ListIterator returns an iterator which yields key-value pairs
// traversing the node.
// If the node kind is anything other than a list, nil will be returned.
//
// The iterator will yield every entry in the list; that is, it
// can be expected that itr.Next will be called node.Length times
// before itr.Done becomes true.
func (n UnixFSBasicDir) ListIterator() ipld.ListIterator {
	return nil
}

// Length returns the length of a list, or the number of entries in a map,
// or -1 if the node is not of list nor map kind.
func (n UnixFSBasicDir) Length() int64 {
	return n._substrate.FieldLinks().Length()
}

func (n UnixFSBasicDir) IsAbsent() bool {
	return false
}

func (n UnixFSBasicDir) IsNull() bool {
	return false
}

func (n UnixFSBasicDir) AsBool() (bool, error) {
	return n._substrate.AsBool()
}

func (n UnixFSBasicDir) AsInt() (int64, error) {
	return n._substrate.AsInt()
}

func (n UnixFSBasicDir) AsFloat() (float64, error) {
	return n._substrate.AsFloat()
}

func (n UnixFSBasicDir) AsString() (string, error) {
	return n._substrate.AsString()
}

func (n UnixFSBasicDir) AsBytes() ([]byte, error) {
	return n._substrate.AsBytes()
}

func (n UnixFSBasicDir) AsLink() (ipld.Link, error) {
	return n._substrate.AsLink()
}

func (n UnixFSBasicDir) Prototype() ipld.NodePrototype {
	// TODO: should this return something?
	// probobly not until we write the write interfaces
	return nil
}

// satisfy schema.TypedNode
func (UnixFSBasicDir) Type() schema.Type {
	return nil /*TODO:typelit*/
}

func (n UnixFSBasicDir) Representation() ipld.Node {
	return n._substrate.Representation()
}

// Native map accessors

func (n UnixFSBasicDir) Iterator() *iter.UnixFSDir__Itr {

	return iter.NewUnixFSDirIterator(n._substrate.Links.Iterator(), nil)
}

func (n UnixFSBasicDir) Lookup(key dagpb.String) dagpb.Link {
	return utils.Lookup(n._substrate.FieldLinks(), key.String())
}

// direct access to the links and data

func (n UnixFSBasicDir) FieldLinks() dagpb.PBLinks {
	return n._substrate.FieldLinks()
}

func (n UnixFSBasicDir) FieldData() dagpb.MaybeBytes {
	return n._substrate.FieldData()
}

// Substrate returns the underlying PBNode -- note: only the substrate will encode successfully to protobuf if writing
func (n UnixFSBasicDir) Substrate() ipld.Node {
	return n._substrate
}
