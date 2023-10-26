package ipldgit

import (
	"bufio"
	"fmt"
	"io"

	"github.com/ipld/go-ipld-prime"
)

// DecodeBlob fills a NodeAssembler (from `Type.Blob__Repr.NewBuilder()`) from a stream of bytes
func DecodeBlob(na ipld.NodeAssembler, rd *bufio.Reader) error {
	sizen, err := readNullTerminatedNumber(rd)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf("blob %d\x00", sizen)
	buf := make([]byte, len(prefix)+sizen)
	copy(buf, prefix)

	n, err := io.ReadFull(rd, buf[len(prefix):])
	if err != nil {
		return err
	}

	if n != sizen {
		return fmt.Errorf("blob size was not accurate")
	}

	return na.AssignBytes(buf)
}

func encodeBlob(n ipld.Node, w io.Writer) error {
	b, err := n.AsBytes()
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}
