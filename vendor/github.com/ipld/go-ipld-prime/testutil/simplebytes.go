package testutil

import (
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/node/mixins"
)

var _ datamodel.Node = simpleBytes(nil)

// simpleBytes is like basicnode's plainBytes but it doesn't implement
// LargeBytesNode so we can exercise the non-LBN case.
type simpleBytes []byte

// NewSimpleBytes is identical to basicnode.NewBytes but the returned node
// doesn't implement LargeBytesNode, which can be useful for testing cases
// where we want to exercise non-LBN code paths.
func NewSimpleBytes(value []byte) datamodel.Node {
	v := simpleBytes(value)
	return &v
}

// -- Node interface methods -->

func (simpleBytes) Kind() datamodel.Kind {
	return datamodel.Kind_Bytes
}
func (simpleBytes) LookupByString(string) (datamodel.Node, error) {
	return mixins.Bytes{TypeName: "bytes"}.LookupByString("")
}
func (simpleBytes) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
	return mixins.Bytes{TypeName: "bytes"}.LookupByNode(nil)
}
func (simpleBytes) LookupByIndex(idx int64) (datamodel.Node, error) {
	return mixins.Bytes{TypeName: "bytes"}.LookupByIndex(0)
}
func (simpleBytes) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return mixins.Bytes{TypeName: "bytes"}.LookupBySegment(seg)
}
func (simpleBytes) MapIterator() datamodel.MapIterator {
	return nil
}
func (simpleBytes) ListIterator() datamodel.ListIterator {
	return nil
}
func (simpleBytes) Length() int64 {
	return -1
}
func (simpleBytes) IsAbsent() bool {
	return false
}
func (simpleBytes) IsNull() bool {
	return false
}
func (simpleBytes) AsBool() (bool, error) {
	return mixins.Bytes{TypeName: "bytes"}.AsBool()
}
func (simpleBytes) AsInt() (int64, error) {
	return mixins.Bytes{TypeName: "bytes"}.AsInt()
}
func (simpleBytes) AsFloat() (float64, error) {
	return mixins.Bytes{TypeName: "bytes"}.AsFloat()
}
func (simpleBytes) AsString() (string, error) {
	return mixins.Bytes{TypeName: "bytes"}.AsString()
}
func (n simpleBytes) AsBytes() ([]byte, error) {
	return []byte(n), nil
}
func (simpleBytes) AsLink() (datamodel.Link, error) {
	return mixins.Bytes{TypeName: "bytes"}.AsLink()
}
func (simpleBytes) Prototype() datamodel.NodePrototype {
	return basicnode.Prototype__Bytes{}
}
