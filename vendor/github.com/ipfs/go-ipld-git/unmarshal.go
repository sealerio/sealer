package ipldgit

import (
	"bufio"
	"fmt"
	"io"

	"github.com/ipld/go-ipld-prime"
)

// Decode reads from a reader to fill a NodeAssembler
func Decode(na ipld.NodeAssembler, r io.Reader) error {
	rd := bufio.NewReader(r)

	typ, err := rd.ReadString(' ')
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	if err != nil {
		return err
	}
	typ = typ[:len(typ)-1]

	switch typ {
	case "tree":
		return DecodeTree(na, rd)
	case "commit":
		return DecodeCommit(na, rd)
	case "blob":
		return DecodeBlob(na, rd)
	case "tag":
		return DecodeTag(na, rd)
	default:
		return fmt.Errorf("unrecognized object type: %q", typ)
	}
}

// ParseObject produces an ipld.Node from a stream / binary represnetation.
func ParseObject(r io.Reader) (ipld.Node, error) {
	rd := bufio.NewReader(r)

	typ, err := rd.ReadString(' ')
	if err == io.EOF {
		return nil, io.ErrUnexpectedEOF
	}
	if err != nil {
		return nil, err
	}
	typ = typ[:len(typ)-1]

	var na ipld.NodeBuilder
	var decode func(ipld.NodeAssembler, *bufio.Reader) error
	switch typ {
	case "tree":
		na = Type.Tree.NewBuilder()
		decode = DecodeTree
	case "commit":
		na = Type.Commit.NewBuilder()
		decode = DecodeCommit
	case "blob":
		na = Type.Blob.NewBuilder()
		decode = DecodeBlob
	case "tag":
		na = Type.Tag.NewBuilder()
		decode = DecodeTag
	default:
		return nil, fmt.Errorf("unrecognized object type: %q", typ)
	}
	// fmt.Printf("type %s\n", typ)

	if err := decode(na, rd); err != nil {
		return nil, err
	}
	return na.Build(), nil
}
