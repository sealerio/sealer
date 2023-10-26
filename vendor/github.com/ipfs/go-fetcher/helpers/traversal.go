package helpers

import (
	"context"

	"github.com/ipfs/go-fetcher"
	"github.com/ipld/go-ipld-prime"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"
)

var matchAllSelector ipld.Node

func init() {
	ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Any)
	matchAllSelector = ssb.ExploreRecursive(selector.RecursionLimitNone(), ssb.ExploreUnion(
		ssb.Matcher(),
		ssb.ExploreAll(ssb.ExploreRecursiveEdge()),
	)).Node()
}

// Block fetches a schemaless node graph corresponding to single block by link.
func Block(ctx context.Context, f fetcher.Fetcher, link ipld.Link) (ipld.Node, error) {
	prototype, err := f.PrototypeFromLink(link)
	if err != nil {
		return nil, err
	}
	return f.BlockOfType(ctx, link, prototype)
}

// BlockMatching traverses a schemaless node graph starting with the given link using the given selector and possibly crossing
// block boundaries. Each matched node is sent to the FetchResult channel.
func BlockMatching(ctx context.Context, f fetcher.Fetcher, root ipld.Link, match ipld.Node, cb fetcher.FetchCallback) error {
	return f.BlockMatchingOfType(ctx, root, match, nil, cb)
}

// BlockAll traverses all nodes in the graph linked by root. The nodes will be untyped and send over the results
// channel.
func BlockAll(ctx context.Context, f fetcher.Fetcher, root ipld.Link, cb fetcher.FetchCallback) error {
	return f.BlockMatchingOfType(ctx, root, matchAllSelector, nil, cb)
}
