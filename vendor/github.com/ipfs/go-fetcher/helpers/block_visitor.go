package helpers

import (
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-fetcher"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

// BlockResult specifies a node at the top of a block boundary
type BlockResult struct {
	Node ipld.Node
	Link ipld.Link
}

// BlockCallback is a callback for visiting blocks
type BlockCallback func(BlockResult) error

// OnBlocks produces a fetch call back that only gets called when visiting blocks during a fetch
func OnBlocks(bv BlockCallback) fetcher.FetchCallback {
	return func(fr fetcher.FetchResult) error {
		if fr.LastBlockPath.String() == fr.Path.String() {
			return bv(BlockResult{
				Node: fr.Node,
				Link: fr.LastBlockLink,
			})
		}
		return nil
	}
}

// OnUniqueBlocks is a callback that only gets called visiting each block once
func OnUniqueBlocks(bv BlockCallback) fetcher.FetchCallback {
	set := cid.NewSet()
	return OnBlocks(func(br BlockResult) error {
		c := br.Link.(cidlink.Link).Cid
		if set.Has(c) {
			return nil
		}
		set.Add(c)
		return bv(br)
	})
}
