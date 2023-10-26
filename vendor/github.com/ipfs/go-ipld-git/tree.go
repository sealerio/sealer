package ipldgit

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// DecodeTree fills a NodeAssembler (from `Type.Tree__Repr.NewBuilder()`) from a stream of bytes
func DecodeTree(na ipld.NodeAssembler, rd *bufio.Reader) error {
	if _, err := readNullTerminatedNumber(rd); err != nil {
		return err
	}

	t := Type.Tree__Repr.NewBuilder()
	ma, err := t.BeginMap(-1)
	if err != nil {
		return err
	}
	for {
		name, node, err := DecodeTreeEntry(rd)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		ee, err := ma.AssembleEntry(name)
		if err != nil {
			return err
		}
		if err = ee.AssignNode(node); err != nil {
			return err
		}
	}
	if err := ma.Finish(); err != nil {
		return err
	}
	return na.AssignNode(t.Build())
}

// DecodeTreeEntry fills a NodeAssembler (from `Type.TreeEntry__Repr.NewBuilder()`) from a stream of bytes
func DecodeTreeEntry(rd *bufio.Reader) (string, ipld.Node, error) {
	data, err := rd.ReadString(' ')
	if err != nil {
		return "", nil, err
	}
	data = data[:len(data)-1]

	name, err := rd.ReadString(0)
	if err != nil {
		return "", nil, err
	}
	name = name[:len(name)-1]

	sha := make([]byte, 20)
	_, err = io.ReadFull(rd, sha)
	if err != nil {
		return "", nil, err
	}

	te := _TreeEntry{
		mode: _String{data},
		hash: _Link{cidlink.Link{Cid: shaToCid(sha)}},
	}
	return name, &te, nil
}

func encodeTree(n ipld.Node, w io.Writer) error {
	buf := new(bytes.Buffer)

	mi := n.MapIterator()
	for !mi.Done() {
		key, te, err := mi.Next()
		if err != nil {
			return err
		}
		name, err := key.AsString()
		if err != nil {
			return err
		}
		if err := encodeTreeEntry(name, te, buf); err != nil {
			return err
		}
	}
	cnt := buf.Len()
	if _, err := fmt.Fprintf(w, "tree %d\x00", cnt); err != nil {
		return err
	}

	_, err := buf.WriteTo(w)
	return err
}

func encodeTreeEntry(name string, n ipld.Node, w io.Writer) error {
	m, err := n.LookupByString("mode")
	if err != nil {
		return err
	}
	ms, err := m.AsString()
	if err != nil {
		return err
	}
	ha, err := n.LookupByString("hash")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s %s\x00", ms, name)
	if err != nil {
		return err
	}

	hal, err := ha.AsLink()
	if err != nil {
		return err
	}
	_, err = w.Write(cidToSha(hal.(cidlink.Link).Cid))
	if err != nil {
		return err
	}

	return nil
}
