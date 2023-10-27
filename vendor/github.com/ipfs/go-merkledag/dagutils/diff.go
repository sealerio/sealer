package dagutils

import (
	"context"
	"fmt"
	"path"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"

	dag "github.com/ipfs/go-merkledag"
)

// ChangeType denotes type of change in Change
type ChangeType int

// These constants define the changes that can be applied to a DAG.
const (
	Add ChangeType = iota
	Remove
	Mod
)

// Change represents a change to a DAG and contains a reference to the old and
// new CIDs.
type Change struct {
	Type   ChangeType
	Path   string
	Before cid.Cid
	After  cid.Cid
}

// String prints a human-friendly line about a change.
func (c *Change) String() string {
	switch c.Type {
	case Add:
		return fmt.Sprintf("Added %s at %s", c.After.String(), c.Path)
	case Remove:
		return fmt.Sprintf("Removed %s from %s", c.Before.String(), c.Path)
	case Mod:
		return fmt.Sprintf("Changed %s to %s at %s", c.Before.String(), c.After.String(), c.Path)
	default:
		panic("nope")
	}
}

// ApplyChange applies the requested changes to the given node in the given dag.
func ApplyChange(ctx context.Context, ds ipld.DAGService, nd *dag.ProtoNode, cs []*Change) (*dag.ProtoNode, error) {
	e := NewDagEditor(nd, ds)
	for _, c := range cs {
		switch c.Type {
		case Add:
			child, err := ds.Get(ctx, c.After)
			if err != nil {
				return nil, err
			}

			childpb, ok := child.(*dag.ProtoNode)
			if !ok {
				return nil, dag.ErrNotProtobuf
			}

			err = e.InsertNodeAtPath(ctx, c.Path, childpb, nil)
			if err != nil {
				return nil, err
			}

		case Remove:
			err := e.RmLink(ctx, c.Path)
			if err != nil {
				return nil, err
			}

		case Mod:
			err := e.RmLink(ctx, c.Path)
			if err != nil {
				return nil, err
			}
			child, err := ds.Get(ctx, c.After)
			if err != nil {
				return nil, err
			}

			childpb, ok := child.(*dag.ProtoNode)
			if !ok {
				return nil, dag.ErrNotProtobuf
			}

			err = e.InsertNodeAtPath(ctx, c.Path, childpb, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	return e.Finalize(ctx, ds)
}

// Diff returns a set of changes that transform node 'a' into node 'b'.
// It only traverses links in the following cases:
// 1. two node's links number are greater than 0.
// 2. both of two nodes are ProtoNode.
// Otherwise, it compares the cid and emits a Mod change object.
func Diff(ctx context.Context, ds ipld.DAGService, a, b ipld.Node) ([]*Change, error) {
	if a.Cid() == b.Cid() {
		return []*Change{}, nil
	}

	cleanA, okA := a.Copy().(*dag.ProtoNode)
	cleanB, okB := b.Copy().(*dag.ProtoNode)

	linksA := a.Links()
	linksB := b.Links()

	if !okA || !okB || (len(linksA) == 0 && len(linksB) == 0) {
		return []*Change{{Type: Mod, Before: a.Cid(), After: b.Cid()}}, nil
	}

	var out []*Change
	for _, linkA := range linksA {
		linkB, _, err := b.ResolveLink([]string{linkA.Name})
		if err != nil {
			continue
		}

		cleanA.RemoveNodeLink(linkA.Name)
		cleanB.RemoveNodeLink(linkA.Name)

		if linkA.Cid == linkB.Cid {
			continue
		}

		nodeA, err := linkA.GetNode(ctx, ds)
		if err != nil {
			return nil, err
		}

		nodeB, err := linkB.GetNode(ctx, ds)
		if err != nil {
			return nil, err
		}

		sub, err := Diff(ctx, ds, nodeA, nodeB)
		if err != nil {
			return nil, err
		}

		for _, c := range sub {
			c.Path = path.Join(linkA.Name, c.Path)
		}

		out = append(out, sub...)
	}

	for _, l := range cleanA.Links() {
		out = append(out, &Change{Type: Remove, Path: l.Name, Before: l.Cid})
	}

	for _, l := range cleanB.Links() {
		out = append(out, &Change{Type: Add, Path: l.Name, After: l.Cid})
	}

	return out, nil
}

// Conflict represents two incompatible changes and is returned by MergeDiffs().
type Conflict struct {
	A *Change
	B *Change
}

// MergeDiffs takes two slice of changes and adds them to a single slice.
// When a Change from b happens to the same path of an existing change in a,
// a conflict is created and b is not added to the merged slice.
// A slice of Conflicts is returned and contains pointers to the
// Changes involved (which share the same path).
func MergeDiffs(a, b []*Change) ([]*Change, []Conflict) {
	paths := make(map[string]*Change)
	for _, c := range b {
		paths[c.Path] = c
	}

	var changes []*Change
	var conflicts []Conflict

	// NOTE: we avoid iterating over maps here to ensure iteration order is determistic. We
	// include changes from a first, then b.
	for _, changeA := range a {
		if changeB, ok := paths[changeA.Path]; ok {
			conflicts = append(conflicts, Conflict{changeA, changeB})
		} else {
			changes = append(changes, changeA)
		}
		delete(paths, changeA.Path)
	}

	for _, c := range b {
		if _, ok := paths[c.Path]; ok {
			changes = append(changes, c)
		}
	}

	return changes, conflicts
}
