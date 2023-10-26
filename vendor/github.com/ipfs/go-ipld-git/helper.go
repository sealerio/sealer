package ipldgit

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"strconv"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	mh "github.com/multiformats/go-multihash"
)

// DecodeBlock attempts to parse a serialized ipfs block into an ipld node dag
// Deprecated: Parse ifrom data instead.
func DecodeBlock(block blocks.Block) (ipld.Node, error) {
	prefix := block.Cid().Prefix()

	if prefix.Codec != cid.GitRaw || prefix.MhType != mh.SHA1 || prefix.MhLength != mh.DefaultLengths[mh.SHA1] {
		return nil, errors.New("invalid CID prefix")
	}

	return ParseObjectFromBuffer(block.RawData())
}

// ParseCompressedObject works like ParseObject, but with a surrounding zlib compression.
func ParseCompressedObject(r io.Reader) (ipld.Node, error) {
	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return ParseObject(rc)
}

// ParseObjectFromBuffer is like ParseObject, but with a fully in-memory stream
func ParseObjectFromBuffer(b []byte) (ipld.Node, error) {
	return ParseObject(bytes.NewReader(b))
}

func readNullTerminatedNumber(rd *bufio.Reader) (int, error) {
	lstr, err := rd.ReadString(0)
	if err != nil {
		return 0, err
	}
	lstr = lstr[:len(lstr)-1]

	return strconv.Atoi(lstr)
}
