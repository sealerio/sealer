package storeutil

import (
	"bytes"
	"fmt"
	"io"

	blocks "github.com/ipfs/go-block-format"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	ipld "github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// LinkSystemForBlockstore constructs an IPLD LinkSystem for a blockstore
func LinkSystemForBlockstore(bs bstore.Blockstore) ipld.LinkSystem {
	lsys := cidlink.DefaultLinkSystem()
	lsys.TrustedStorage = true
	lsys.StorageReadOpener = func(lnkCtx ipld.LinkContext, lnk ipld.Link) (io.Reader, error) {
		asCidLink, ok := lnk.(cidlink.Link)
		if !ok {
			return nil, fmt.Errorf("unsupported link type")
		}

		block, err := bs.Get(lnkCtx.Ctx, asCidLink.Cid)
		if err != nil {
			return nil, err
		}
		return bytes.NewBuffer(block.RawData()), nil
	}
	lsys.StorageWriteOpener = func(lnkCtx ipld.LinkContext) (io.Writer, ipld.BlockWriteCommitter, error) {
		var buffer settableBuffer
		committer := func(lnk ipld.Link) error {
			asCidLink, ok := lnk.(cidlink.Link)
			if !ok {
				return fmt.Errorf("unsupported link type")
			}
			block, err := blocks.NewBlockWithCid(buffer.Bytes(), asCidLink.Cid)
			if err != nil {
				return err
			}
			return bs.Put(lnkCtx.Ctx, block)
		}
		return &buffer, committer, nil
	}
	return lsys
}

type settableBuffer struct {
	bytes.Buffer
	didSetData bool
	data       []byte
}

func (sb *settableBuffer) SetBytes(data []byte) error {
	sb.didSetData = true
	sb.data = data
	return nil
}

func (sb *settableBuffer) Bytes() []byte {
	if sb.didSetData {
		return sb.data
	}
	return sb.Buffer.Bytes()
}
