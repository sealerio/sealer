package io

import (
	"context"
	"fmt"
	"os"

	"github.com/ipfs/go-unixfs/hamt"
	"github.com/ipfs/go-unixfs/private/linksize"

	"github.com/alecthomas/units"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log"
	mdag "github.com/ipfs/go-merkledag"
	format "github.com/ipfs/go-unixfs"
)

var log = logging.Logger("unixfs")

// HAMTShardingSize is a global option that allows switching to a HAMTDirectory
// when the BasicDirectory grows above the size (in bytes) signalled by this
// flag. The default size of 0 disables the option.
// The size is not the *exact* block size of the encoded BasicDirectory but just
// the estimated size based byte length of links name and CID (BasicDirectory's
// ProtoNode doesn't use the Data field so this estimate is pretty accurate).
var HAMTShardingSize = int(256 * units.KiB)

// DefaultShardWidth is the default value used for hamt sharding width.
// Needs to be a power of two (shard entry size) and multiple of 8 (bitfield size).
var DefaultShardWidth = 256

// Directory defines a UnixFS directory. It is used for creating, reading and
// editing directories. It allows to work with different directory schemes,
// like the basic or the HAMT implementation.
//
// It just allows to perform explicit edits on a single directory, working with
// directory trees is out of its scope, they are managed by the MFS layer
// (which is the main consumer of this interface).
type Directory interface {

	// SetCidBuilder sets the CID Builder of the root node.
	SetCidBuilder(cid.Builder)

	// AddChild adds a (name, key) pair to the root node.
	AddChild(context.Context, string, ipld.Node) error

	// ForEachLink applies the given function to Links in the directory.
	ForEachLink(context.Context, func(*ipld.Link) error) error

	// EnumLinksAsync returns a channel which will receive Links in the directory
	// as they are enumerated, where order is not gauranteed
	EnumLinksAsync(context.Context) <-chan format.LinkResult

	// Links returns the all the links in the directory node.
	Links(context.Context) ([]*ipld.Link, error)

	// Find returns the root node of the file named 'name' within this directory.
	// In the case of HAMT-directories, it will traverse the tree.
	//
	// Returns os.ErrNotExist if the child does not exist.
	Find(context.Context, string) (ipld.Node, error)

	// RemoveChild removes the child with the given name.
	//
	// Returns os.ErrNotExist if the child doesn't exist.
	RemoveChild(context.Context, string) error

	// GetNode returns the root of this directory.
	GetNode() (ipld.Node, error)

	// GetCidBuilder returns the CID Builder used.
	GetCidBuilder() cid.Builder
}

// TODO: Evaluate removing `dserv` from this layer and providing it in MFS.
// (The functions should in that case add a `DAGService` argument.)

// Link size estimation function. For production it's usually the one here
// but during test we may mock it to get fixed sizes.
func productionLinkSize(linkName string, linkCid cid.Cid) int {
	return len(linkName) + linkCid.ByteLen()
}

func init() {
	linksize.LinkSizeFunction = productionLinkSize
}

// BasicDirectory is the basic implementation of `Directory`. All the entries
// are stored in a single node.
type BasicDirectory struct {
	node  *mdag.ProtoNode
	dserv ipld.DAGService

	// Internal variable used to cache the estimated size of the basic directory:
	// for each link, aggregate link name + link CID. DO NOT CHANGE THIS
	// as it will affect the HAMT transition behavior in HAMTShardingSize.
	// (We maintain this value up to date even if the HAMTShardingSize is off
	// since potentially the option could be activated on the fly.)
	estimatedSize int
}

// HAMTDirectory is the HAMT implementation of `Directory`.
// (See package `hamt` for more information.)
type HAMTDirectory struct {
	shard *hamt.Shard
	dserv ipld.DAGService

	// Track the changes in size by the AddChild and RemoveChild calls
	// for the HAMTShardingSize option.
	sizeChange int
}

func newEmptyBasicDirectory(dserv ipld.DAGService) *BasicDirectory {
	return newBasicDirectoryFromNode(dserv, format.EmptyDirNode())
}

func newBasicDirectoryFromNode(dserv ipld.DAGService, node *mdag.ProtoNode) *BasicDirectory {
	basicDir := new(BasicDirectory)
	basicDir.node = node
	basicDir.dserv = dserv

	// Scan node links (if any) to restore estimated size.
	basicDir.computeEstimatedSize()

	return basicDir
}

// NewDirectory returns a Directory implemented by DynamicDirectory
// containing a BasicDirectory that can be converted to a HAMTDirectory.
func NewDirectory(dserv ipld.DAGService) Directory {
	return &DynamicDirectory{newEmptyBasicDirectory(dserv)}
}

// ErrNotADir implies that the given node was not a unixfs directory
var ErrNotADir = fmt.Errorf("merkledag node was not a directory or shard")

// NewDirectoryFromNode loads a unixfs directory from the given IPLD node and
// DAGService.
func NewDirectoryFromNode(dserv ipld.DAGService, node ipld.Node) (Directory, error) {
	protoBufNode, ok := node.(*mdag.ProtoNode)
	if !ok {
		return nil, ErrNotADir
	}

	fsNode, err := format.FSNodeFromBytes(protoBufNode.Data())
	if err != nil {
		return nil, err
	}

	switch fsNode.Type() {
	case format.TDirectory:
		return &DynamicDirectory{newBasicDirectoryFromNode(dserv, protoBufNode.Copy().(*mdag.ProtoNode))}, nil
	case format.THAMTShard:
		shard, err := hamt.NewHamtFromDag(dserv, node)
		if err != nil {
			return nil, err
		}
		return &DynamicDirectory{&HAMTDirectory{shard, dserv, 0}}, nil
	}

	return nil, ErrNotADir
}

func (d *BasicDirectory) computeEstimatedSize() {
	d.estimatedSize = 0
	d.ForEachLink(context.TODO(), func(l *ipld.Link) error {
		d.addToEstimatedSize(l.Name, l.Cid)
		return nil
	})
	// ForEachLink will never fail traversing the BasicDirectory
	// and neither the inner callback `addToEstimatedSize`.
}

func (d *BasicDirectory) addToEstimatedSize(name string, linkCid cid.Cid) {
	d.estimatedSize += linksize.LinkSizeFunction(name, linkCid)
}

func (d *BasicDirectory) removeFromEstimatedSize(name string, linkCid cid.Cid) {
	d.estimatedSize -= linksize.LinkSizeFunction(name, linkCid)
	if d.estimatedSize < 0 {
		// Something has gone very wrong. Log an error and recompute the
		// size from scratch.
		log.Error("BasicDirectory's estimatedSize went below 0")
		d.computeEstimatedSize()
	}
}

// SetCidBuilder implements the `Directory` interface.
func (d *BasicDirectory) SetCidBuilder(builder cid.Builder) {
	d.node.SetCidBuilder(builder)
}

// AddChild implements the `Directory` interface. It adds (or replaces)
// a link to the given `node` under `name`.
func (d *BasicDirectory) AddChild(ctx context.Context, name string, node ipld.Node) error {
	link, err := ipld.MakeLink(node)
	if err != nil {
		return err
	}

	return d.addLinkChild(ctx, name, link)
}

func (d *BasicDirectory) needsToSwitchToHAMTDir(name string, nodeToAdd ipld.Node) (bool, error) {
	if HAMTShardingSize == 0 { // Option disabled.
		return false, nil
	}

	operationSizeChange := 0
	// Find if there is an old entry under that name that will be overwritten.
	entryToRemove, err := d.node.GetNodeLink(name)
	if err != mdag.ErrLinkNotFound {
		if err != nil {
			return false, err
		}
		operationSizeChange -= linksize.LinkSizeFunction(name, entryToRemove.Cid)
	}
	if nodeToAdd != nil {
		operationSizeChange += linksize.LinkSizeFunction(name, nodeToAdd.Cid())
	}

	return d.estimatedSize+operationSizeChange >= HAMTShardingSize, nil
}

// addLinkChild adds the link as an entry to this directory under the given
// name. Plumbing function for the AddChild API.
func (d *BasicDirectory) addLinkChild(ctx context.Context, name string, link *ipld.Link) error {
	// Remove old link and account for size change (if it existed; ignore
	// `ErrNotExist` otherwise).
	err := d.RemoveChild(ctx, name)
	if err != nil && err != os.ErrNotExist {
		return err
	}

	err = d.node.AddRawLink(name, link)
	if err != nil {
		return err
	}
	d.addToEstimatedSize(name, link.Cid)
	return nil
}

// EnumLinksAsync returns a channel which will receive Links in the directory
// as they are enumerated, where order is not gauranteed
func (d *BasicDirectory) EnumLinksAsync(ctx context.Context) <-chan format.LinkResult {
	linkResults := make(chan format.LinkResult)
	go func() {
		defer close(linkResults)
		for _, l := range d.node.Links() {
			select {
			case linkResults <- format.LinkResult{
				Link: l,
				Err:  nil,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return linkResults
}

// ForEachLink implements the `Directory` interface.
func (d *BasicDirectory) ForEachLink(_ context.Context, f func(*ipld.Link) error) error {
	for _, l := range d.node.Links() {
		if err := f(l); err != nil {
			return err
		}
	}
	return nil
}

// Links implements the `Directory` interface.
func (d *BasicDirectory) Links(ctx context.Context) ([]*ipld.Link, error) {
	return d.node.Links(), nil
}

// Find implements the `Directory` interface.
func (d *BasicDirectory) Find(ctx context.Context, name string) (ipld.Node, error) {
	lnk, err := d.node.GetNodeLink(name)
	if err == mdag.ErrLinkNotFound {
		err = os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}

	return d.dserv.Get(ctx, lnk.Cid)
}

// RemoveChild implements the `Directory` interface.
func (d *BasicDirectory) RemoveChild(ctx context.Context, name string) error {
	// We need to *retrieve* the link before removing it to update the estimated
	// size. This means we may iterate the links slice twice: if traversing this
	// becomes a problem, a factor of 2 isn't going to make much of a difference.
	// We'd likely need to cache a link resolution map in that case.
	link, err := d.node.GetNodeLink(name)
	if err == mdag.ErrLinkNotFound {
		return os.ErrNotExist
	}
	if err != nil {
		return err // at the moment there is no other error besides ErrLinkNotFound
	}

	// The name actually existed so we should update the estimated size.
	d.removeFromEstimatedSize(link.Name, link.Cid)

	return d.node.RemoveNodeLink(name)
	// GetNodeLink didn't return ErrLinkNotFound so this won't fail with that
	// and we don't need to convert the error again.
}

// GetNode implements the `Directory` interface.
func (d *BasicDirectory) GetNode() (ipld.Node, error) {
	return d.node, nil
}

// GetCidBuilder implements the `Directory` interface.
func (d *BasicDirectory) GetCidBuilder() cid.Builder {
	return d.node.CidBuilder()
}

// switchToSharding returns a HAMT implementation of this directory.
func (d *BasicDirectory) switchToSharding(ctx context.Context) (*HAMTDirectory, error) {
	hamtDir := new(HAMTDirectory)
	hamtDir.dserv = d.dserv

	shard, err := hamt.NewShard(d.dserv, DefaultShardWidth)
	if err != nil {
		return nil, err
	}
	shard.SetCidBuilder(d.node.CidBuilder())
	hamtDir.shard = shard

	for _, lnk := range d.node.Links() {
		node, err := d.dserv.Get(ctx, lnk.Cid)
		if err != nil {
			return nil, err
		}

		err = hamtDir.shard.Set(ctx, lnk.Name, node)
		if err != nil {
			return nil, err
		}
	}

	return hamtDir, nil
}

// SetCidBuilder implements the `Directory` interface.
func (d *HAMTDirectory) SetCidBuilder(builder cid.Builder) {
	d.shard.SetCidBuilder(builder)
}

// AddChild implements the `Directory` interface.
func (d *HAMTDirectory) AddChild(ctx context.Context, name string, nd ipld.Node) error {
	oldChild, err := d.shard.Swap(ctx, name, nd)
	if err != nil {
		return err
	}

	if oldChild != nil {
		d.removeFromSizeChange(oldChild.Name, oldChild.Cid)
	}
	d.addToSizeChange(name, nd.Cid())
	return nil
}

// ForEachLink implements the `Directory` interface.
func (d *HAMTDirectory) ForEachLink(ctx context.Context, f func(*ipld.Link) error) error {
	return d.shard.ForEachLink(ctx, f)
}

// EnumLinksAsync returns a channel which will receive Links in the directory
// as they are enumerated, where order is not gauranteed
func (d *HAMTDirectory) EnumLinksAsync(ctx context.Context) <-chan format.LinkResult {
	return d.shard.EnumLinksAsync(ctx)
}

// Links implements the `Directory` interface.
func (d *HAMTDirectory) Links(ctx context.Context) ([]*ipld.Link, error) {
	return d.shard.EnumLinks(ctx)
}

// Find implements the `Directory` interface. It will traverse the tree.
func (d *HAMTDirectory) Find(ctx context.Context, name string) (ipld.Node, error) {
	lnk, err := d.shard.Find(ctx, name)
	if err != nil {
		return nil, err
	}

	return lnk.GetNode(ctx, d.dserv)
}

// RemoveChild implements the `Directory` interface.
func (d *HAMTDirectory) RemoveChild(ctx context.Context, name string) error {
	oldChild, err := d.shard.Take(ctx, name)
	if err != nil {
		return err
	}

	if oldChild != nil {
		d.removeFromSizeChange(oldChild.Name, oldChild.Cid)
	}

	return nil
}

// GetNode implements the `Directory` interface.
func (d *HAMTDirectory) GetNode() (ipld.Node, error) {
	return d.shard.Node()
}

// GetCidBuilder implements the `Directory` interface.
func (d *HAMTDirectory) GetCidBuilder() cid.Builder {
	return d.shard.CidBuilder()
}

// switchToBasic returns a BasicDirectory implementation of this directory.
func (d *HAMTDirectory) switchToBasic(ctx context.Context) (*BasicDirectory, error) {
	basicDir := newEmptyBasicDirectory(d.dserv)
	basicDir.SetCidBuilder(d.GetCidBuilder())

	err := d.ForEachLink(ctx, func(lnk *ipld.Link) error {
		err := basicDir.addLinkChild(ctx, lnk.Name, lnk)
		if err != nil {
			return err
		}

		return nil
		// This function enumerates all the links in the Directory requiring all
		// shards to be accessible but it is only called *after* sizeBelowThreshold
		// returns true, which means we have already enumerated and fetched *all*
		// shards in the first place (that's the only way we can be really sure
		// we are actually below the threshold).
	})
	if err != nil {
		return nil, err
	}

	return basicDir, nil
}

func (d *HAMTDirectory) addToSizeChange(name string, linkCid cid.Cid) {
	d.sizeChange += linksize.LinkSizeFunction(name, linkCid)
}

func (d *HAMTDirectory) removeFromSizeChange(name string, linkCid cid.Cid) {
	d.sizeChange -= linksize.LinkSizeFunction(name, linkCid)
}

// Evaluate a switch from HAMTDirectory to BasicDirectory in case the size will
// go above the threshold when we are adding or removing an entry.
// In both the add/remove operations any old name will be removed, and for the
// add operation in particular a new entry will be added under that name (otherwise
// nodeToAdd is nil). We compute both (potential) future subtraction and
// addition to the size change.
func (d *HAMTDirectory) needsToSwitchToBasicDir(ctx context.Context, name string, nodeToAdd ipld.Node) (switchToBasic bool, err error) {
	if HAMTShardingSize == 0 { // Option disabled.
		return false, nil
	}

	operationSizeChange := 0

	// Find if there is an old entry under that name that will be overwritten
	// (AddEntry) or flat out removed (RemoveEntry).
	entryToRemove, err := d.shard.Find(ctx, name)
	if err != os.ErrNotExist {
		if err != nil {
			return false, err
		}
		operationSizeChange -= linksize.LinkSizeFunction(name, entryToRemove.Cid)
	}

	// For the AddEntry case compute the size addition of the new entry.
	if nodeToAdd != nil {
		operationSizeChange += linksize.LinkSizeFunction(name, nodeToAdd.Cid())
	}

	if d.sizeChange+operationSizeChange >= 0 {
		// We won't have reduced the HAMT net size.
		return false, nil
	}

	// We have reduced the directory size, check if went below the
	// HAMTShardingSize threshold to trigger a switch.
	return d.sizeBelowThreshold(ctx, operationSizeChange)
}

// Evaluate directory size and a future sizeChange and check if it will be below
// HAMTShardingSize threshold (to trigger a transition to a BasicDirectory).
// Instead of enumerating the entire tree we eagerly call EnumLinksAsync
// until we either reach a value above the threshold (in that case no need
// to keep counting) or an error occurs (like the context being canceled
// if we take too much time fetching the necessary shards).
func (d *HAMTDirectory) sizeBelowThreshold(ctx context.Context, sizeChange int) (below bool, err error) {
	if HAMTShardingSize == 0 {
		panic("asked to compute HAMT size with HAMTShardingSize option off (0)")
	}

	// We don't necessarily compute the full size of *all* shards as we might
	// end early if we already know we're above the threshold or run out of time.
	partialSize := 0

	// We stop the enumeration once we have enough information and exit this function.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for linkResult := range d.EnumLinksAsync(ctx) {
		if linkResult.Err != nil {
			return false, linkResult.Err
		}

		partialSize += linksize.LinkSizeFunction(linkResult.Link.Name, linkResult.Link.Cid)
		if partialSize+sizeChange >= HAMTShardingSize {
			// We have already fetched enough shards to assert we are
			//  above the threshold, so no need to keep fetching.
			return false, nil
		}
	}

	// We enumerated *all* links in all shards and didn't reach the threshold.
	return true, nil
}

// DynamicDirectory wraps a Directory interface and provides extra logic
// to switch from BasicDirectory to HAMTDirectory and backwards based on
// size.
type DynamicDirectory struct {
	Directory
}

var _ Directory = (*DynamicDirectory)(nil)

// AddChild implements the `Directory` interface. We check when adding new entries
// if we should switch to HAMTDirectory according to global option(s).
func (d *DynamicDirectory) AddChild(ctx context.Context, name string, nd ipld.Node) error {
	hamtDir, ok := d.Directory.(*HAMTDirectory)
	if ok {
		// We evaluate a switch in the HAMTDirectory case even for an AddChild
		// as it may overwrite an existing entry and end up actually reducing
		// the directory size.
		switchToBasic, err := hamtDir.needsToSwitchToBasicDir(ctx, name, nd)
		if err != nil {
			return err
		}

		if switchToBasic {
			basicDir, err := hamtDir.switchToBasic(ctx)
			if err != nil {
				return err
			}
			err = basicDir.AddChild(ctx, name, nd)
			if err != nil {
				return err
			}
			d.Directory = basicDir
			return nil
		}

		return d.Directory.AddChild(ctx, name, nd)
	}

	// BasicDirectory
	basicDir := d.Directory.(*BasicDirectory)
	switchToHAMT, err := basicDir.needsToSwitchToHAMTDir(name, nd)
	if err != nil {
		return err
	}
	if !switchToHAMT {
		return basicDir.AddChild(ctx, name, nd)
	}
	hamtDir, err = basicDir.switchToSharding(ctx)
	if err != nil {
		return err
	}
	hamtDir.AddChild(ctx, name, nd)
	if err != nil {
		return err
	}
	d.Directory = hamtDir
	return nil
}

// RemoveChild implements the `Directory` interface. Used in the case where we wrap
// a HAMTDirectory that might need to be downgraded to a BasicDirectory. The
// upgrade path is in AddChild.
func (d *DynamicDirectory) RemoveChild(ctx context.Context, name string) error {
	hamtDir, ok := d.Directory.(*HAMTDirectory)
	if !ok {
		return d.Directory.RemoveChild(ctx, name)
	}

	switchToBasic, err := hamtDir.needsToSwitchToBasicDir(ctx, name, nil)
	if err != nil {
		return err
	}

	if !switchToBasic {
		return hamtDir.RemoveChild(ctx, name)
	}

	basicDir, err := hamtDir.switchToBasic(ctx)
	if err != nil {
		return err
	}
	basicDir.RemoveChild(ctx, name)
	if err != nil {
		return err
	}
	d.Directory = basicDir
	return nil
}
