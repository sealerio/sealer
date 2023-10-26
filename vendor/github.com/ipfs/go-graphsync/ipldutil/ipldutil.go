package ipldutil

import (
	"bytes"

	dagpb "github.com/ipld/go-codec-dagpb"
	ipld "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	_ "github.com/ipld/go-ipld-prime/codec/raw"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
)

var defaultChooser = func(lnk ipld.Link, lctx ipld.LinkContext) (ipld.NodePrototype, error) {
	// We can decode all nodes into basicnode's Any, except for
	// dagpb nodes, which must explicitly use the PBNode prototype.
	if lnk, ok := lnk.(cidlink.Link); ok && lnk.Cid.Prefix().Codec == 0x70 {
		return dagpb.Type.PBNode, nil
	}
	return basicnode.Prototype.Any, nil
}

func EncodeNode(node ipld.Node) ([]byte, error) {
	var buffer bytes.Buffer
	err := dagcbor.Encode(node, &buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func DecodeNode(encoded []byte) (ipld.Node, error) {
	nb := basicnode.Prototype.Any.NewBuilder()
	if err := dagcbor.Decode(nb, bytes.NewReader(encoded)); err != nil {
		return nil, err
	}
	return nb.Build(), nil
}
