package testutil

import (
	"io"

	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
)

var _ datamodel.Node = MultiByteNode{}
var _ datamodel.LargeBytesNode = (*MultiByteNode)(nil)

// MultiByteNode is a node that is a concatenation of multiple byte slices.
// It's not particularly sophisticated but lets us exercise LargeBytesNode in a
// non-trivial way.
// The novel behaviour of Read() and Seek() on the AsLargeBytes is similar to
// that which would be expected from a LBN ADL, such as UnixFS sharded files.
type MultiByteNode struct {
	bytes [][]byte
}

func NewMultiByteNode(bytes ...[]byte) MultiByteNode {
	return MultiByteNode{bytes: bytes}
}

func (mbn MultiByteNode) Kind() datamodel.Kind {
	return datamodel.Kind_Bytes
}

func (mbn MultiByteNode) AsBytes() ([]byte, error) {
	ret := make([]byte, 0, mbn.TotalLength())
	for _, b := range mbn.bytes {
		ret = append(ret, b...)
	}
	return ret, nil
}

func (mbn MultiByteNode) TotalLength() int {
	var size int
	for _, b := range mbn.bytes {
		size += len(b)
	}
	return size
}

func (mbn MultiByteNode) AsLargeBytes() (io.ReadSeeker, error) {
	return &mbnReadSeeker{node: mbn}, nil
}

func (mbn MultiByteNode) AsBool() (bool, error) {
	return false, datamodel.ErrWrongKind{TypeName: "bool", MethodName: "AsBool", AppropriateKind: datamodel.KindSet_JustBytes}
}

func (mbn MultiByteNode) AsInt() (int64, error) {
	return 0, datamodel.ErrWrongKind{TypeName: "int", MethodName: "AsInt", AppropriateKind: datamodel.KindSet_JustBytes}
}

func (mbn MultiByteNode) AsFloat() (float64, error) {
	return 0, datamodel.ErrWrongKind{TypeName: "float", MethodName: "AsFloat", AppropriateKind: datamodel.KindSet_JustBytes}
}

func (mbn MultiByteNode) AsString() (string, error) {
	return "", datamodel.ErrWrongKind{TypeName: "string", MethodName: "AsString", AppropriateKind: datamodel.KindSet_JustBytes}
}

func (mbn MultiByteNode) AsLink() (datamodel.Link, error) {
	return nil, datamodel.ErrWrongKind{TypeName: "link", MethodName: "AsLink", AppropriateKind: datamodel.KindSet_JustBytes}
}

func (mbn MultiByteNode) AsNode() (datamodel.Node, error) {
	return nil, nil
}

func (mbn MultiByteNode) Size() int {
	return 0
}

func (mbn MultiByteNode) IsAbsent() bool {
	return false
}

func (mbn MultiByteNode) IsNull() bool {
	return false
}

func (mbn MultiByteNode) Length() int64 {
	return 0
}

func (mbn MultiByteNode) ListIterator() datamodel.ListIterator {
	return nil
}

func (mbn MultiByteNode) MapIterator() datamodel.MapIterator {
	return nil
}

func (mbn MultiByteNode) LookupByIndex(idx int64) (datamodel.Node, error) {
	return nil, datamodel.ErrWrongKind{}
}

func (mbn MultiByteNode) LookupByString(key string) (datamodel.Node, error) {
	return nil, datamodel.ErrWrongKind{}
}

func (mbn MultiByteNode) LookupByNode(key datamodel.Node) (datamodel.Node, error) {
	return nil, datamodel.ErrWrongKind{}
}

func (mbn MultiByteNode) LookupBySegment(seg datamodel.PathSegment) (datamodel.Node, error) {
	return nil, datamodel.ErrWrongKind{}
}

func (mbn MultiByteNode) Prototype() datamodel.NodePrototype {
	return basicnode.Prototype.Bytes // not really ... but it'll do for this test
}

type mbnReadSeeker struct {
	node   MultiByteNode
	offset int
}

func (mbnrs *mbnReadSeeker) Read(p []byte) (int, error) {
	var acc int
	for _, byts := range mbnrs.node.bytes {
		if mbnrs.offset-acc >= len(byts) {
			acc += len(byts)
			continue
		}
		n := copy(p, byts[mbnrs.offset-acc:])
		mbnrs.offset += n
		return n, nil
	}
	return 0, io.EOF
}

func (mbnrs *mbnReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		mbnrs.offset = int(offset)
	case io.SeekCurrent:
		mbnrs.offset += int(offset)
	case io.SeekEnd:
		mbnrs.offset = mbnrs.node.TotalLength() + int(offset)
	}
	return int64(mbnrs.offset), nil
}
