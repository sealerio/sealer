package ipldgit

import (
	"fmt"
	"io"

	"github.com/ipld/go-ipld-prime"
)

// Encode serializes a git node to a raw binary form.
func Encode(n ipld.Node, w io.Writer) error {
	switch n.Prototype() {
	case Type.Blob, Type.Blob__Repr:
		return encodeBlob(n, w)
	case Type.Commit, Type.Commit__Repr:
		return encodeCommit(n, w)
	case Type.Tree, Type.Tree__Repr:
		return encodeTree(n, w)
	case Type.Tag, Type.Tag__Repr:
		return encodeTag(n, w)
	default:
	}
	switch n.Kind() {
	case ipld.Kind_Bytes:
		return encodeBlob(n, w)
	case ipld.Kind_List:
		return encodeTree(n, w)
	case ipld.Kind_Map:
		k, _, err := n.MapIterator().Next()
		if err != nil {
			return err
		}
		repKey, err := k.AsString()
		if err != nil {
			return err
		}
		switch repKey {
		case
			"object",
			"type",
			"tag",
			"tagger",
			"text":
			return encodeTag(n, w)
		default:
			return encodeCommit(n, w)
		}
	default:
		return fmt.Errorf("unrecognized object type: %T", n.Prototype())
	}
}
