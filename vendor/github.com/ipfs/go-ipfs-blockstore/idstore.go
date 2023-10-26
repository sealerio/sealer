package blockstore

import (
	"context"

	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// idstore wraps a BlockStore to add support for identity hashes
type idstore struct {
	bs Blockstore
}

func NewIdStore(bs Blockstore) Blockstore {
	return &idstore{bs}
}

func extractContents(k cid.Cid) (bool, []byte) {
	dmh, err := mh.Decode(k.Hash())
	if err != nil || dmh.Code != mh.ID {
		return false, nil
	}
	return true, dmh.Digest
}

func (b *idstore) DeleteBlock(ctx context.Context, k cid.Cid) error {
	isId, _ := extractContents(k)
	if isId {
		return nil
	}
	return b.bs.DeleteBlock(ctx, k)
}

func (b *idstore) Has(ctx context.Context, k cid.Cid) (bool, error) {
	isId, _ := extractContents(k)
	if isId {
		return true, nil
	}
	return b.bs.Has(ctx, k)
}

func (b *idstore) GetSize(ctx context.Context, k cid.Cid) (int, error) {
	isId, bdata := extractContents(k)
	if isId {
		return len(bdata), nil
	}
	return b.bs.GetSize(ctx, k)
}

func (b *idstore) Get(ctx context.Context, k cid.Cid) (blocks.Block, error) {
	isId, bdata := extractContents(k)
	if isId {
		return blocks.NewBlockWithCid(bdata, k)
	}
	return b.bs.Get(ctx, k)
}

func (b *idstore) Put(ctx context.Context, bl blocks.Block) error {
	isId, _ := extractContents(bl.Cid())
	if isId {
		return nil
	}
	return b.bs.Put(ctx, bl)
}

func (b *idstore) PutMany(ctx context.Context, bs []blocks.Block) error {
	toPut := make([]blocks.Block, 0, len(bs))
	for _, bl := range bs {
		isId, _ := extractContents(bl.Cid())
		if isId {
			continue
		}
		toPut = append(toPut, bl)
	}
	return b.bs.PutMany(ctx, toPut)
}

func (b *idstore) HashOnRead(enabled bool) {
	b.bs.HashOnRead(enabled)
}

func (b *idstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return b.bs.AllKeysChan(ctx)
}
