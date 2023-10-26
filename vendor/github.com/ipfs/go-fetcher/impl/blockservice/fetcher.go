package bsfetcher

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-fetcher"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/ipld/go-ipld-prime/traversal/selector"
)

type fetcherSession struct {
	linkSystem   ipld.LinkSystem
	protoChooser traversal.LinkTargetNodePrototypeChooser
}

// FetcherConfig defines a configuration object from which Fetcher instances are constructed
type FetcherConfig struct {
	blockService     blockservice.BlockService
	NodeReifier      ipld.NodeReifier
	PrototypeChooser traversal.LinkTargetNodePrototypeChooser
}

// NewFetcherConfig creates a FetchConfig from which session may be created and nodes retrieved.
func NewFetcherConfig(blockService blockservice.BlockService) FetcherConfig {
	return FetcherConfig{
		blockService:     blockService,
		PrototypeChooser: DefaultPrototypeChooser,
	}
}

// NewSession creates a session from which nodes may be retrieved.
// The session ends when the provided context is canceled.
func (fc FetcherConfig) NewSession(ctx context.Context) fetcher.Fetcher {
	return fc.FetcherWithSession(ctx, blockservice.NewSession(ctx, fc.blockService))
}

func (fc FetcherConfig) FetcherWithSession(ctx context.Context, s *blockservice.Session) fetcher.Fetcher {
	ls := cidlink.DefaultLinkSystem()
	// while we may be loading blocks remotely, they are already hash verified by the time they load
	// into ipld-prime
	ls.TrustedStorage = true
	ls.StorageReadOpener = blockOpener(ctx, s)
	ls.NodeReifier = fc.NodeReifier

	protoChooser := fc.PrototypeChooser
	return &fetcherSession{linkSystem: ls, protoChooser: protoChooser}
}

// WithReifier derives a different fetcher factory from the same source but
// with a chosen NodeReifier for pathing semantics.
func (fc FetcherConfig) WithReifier(nr ipld.NodeReifier) fetcher.Factory {
	return FetcherConfig{
		blockService:     fc.blockService,
		NodeReifier:      nr,
		PrototypeChooser: fc.PrototypeChooser,
	}
}

// interface check
var _ fetcher.Factory = FetcherConfig{}

// BlockOfType fetches a node graph of the provided type corresponding to single block by link.
func (f *fetcherSession) BlockOfType(ctx context.Context, link ipld.Link, ptype ipld.NodePrototype) (ipld.Node, error) {
	return f.linkSystem.Load(ipld.LinkContext{}, link, ptype)
}

func (f *fetcherSession) nodeMatching(ctx context.Context, initialProgress traversal.Progress, node ipld.Node, match ipld.Node, cb fetcher.FetchCallback) error {
	matchSelector, err := selector.ParseSelector(match)
	if err != nil {
		return err
	}
	return initialProgress.WalkMatching(node, matchSelector, func(prog traversal.Progress, n ipld.Node) error {
		return cb(fetcher.FetchResult{
			Node:          n,
			Path:          prog.Path,
			LastBlockPath: prog.LastBlock.Path,
			LastBlockLink: prog.LastBlock.Link,
		})
	})
}

func (f *fetcherSession) blankProgress(ctx context.Context) traversal.Progress {
	return traversal.Progress{
		Cfg: &traversal.Config{
			LinkSystem:                     f.linkSystem,
			LinkTargetNodePrototypeChooser: f.protoChooser,
		},
	}
}

func (f *fetcherSession) NodeMatching(ctx context.Context, node ipld.Node, match ipld.Node, cb fetcher.FetchCallback) error {
	return f.nodeMatching(ctx, f.blankProgress(ctx), node, match, cb)
}

func (f *fetcherSession) BlockMatchingOfType(ctx context.Context, root ipld.Link, match ipld.Node,
	_ ipld.NodePrototype, cb fetcher.FetchCallback) error {

	// retrieve first node
	prototype, err := f.PrototypeFromLink(root)
	if err != nil {
		return err
	}
	node, err := f.BlockOfType(ctx, root, prototype)
	if err != nil {
		return err
	}

	progress := f.blankProgress(ctx)
	progress.LastBlock.Link = root
	return f.nodeMatching(ctx, progress, node, match, cb)
}

func (f *fetcherSession) PrototypeFromLink(lnk ipld.Link) (ipld.NodePrototype, error) {
	return f.protoChooser(lnk, ipld.LinkContext{})
}

// DefaultPrototypeChooser supports choosing the prototype from the link and falling
// back to a basicnode.Any builder
var DefaultPrototypeChooser = func(lnk ipld.Link, lnkCtx ipld.LinkContext) (ipld.NodePrototype, error) {
	if tlnkNd, ok := lnkCtx.LinkNode.(schema.TypedLinkNode); ok {
		return tlnkNd.LinkTargetNodePrototype(), nil
	}
	return basicnode.Prototype.Any, nil
}

func blockOpener(ctx context.Context, bs *blockservice.Session) ipld.BlockReadOpener {
	return func(_ ipld.LinkContext, lnk ipld.Link) (io.Reader, error) {
		cidLink, ok := lnk.(cidlink.Link)
		if !ok {
			return nil, fmt.Errorf("invalid link type for loading: %v", lnk)
		}

		blk, err := bs.GetBlock(ctx, cidLink.Cid)
		if err != nil {
			return nil, err
		}

		return bytes.NewReader(blk.RawData()), nil
	}
}
