// Package wantlist implements an object for bitswap that contains the keys
// that a given peer wants.
package wantlist

import (
	"sort"

	pb "github.com/ipfs/go-bitswap/message/pb"

	cid "github.com/ipfs/go-cid"
)

// Wantlist is a raw list of wanted blocks and their priorities
type Wantlist struct {
	set map[cid.Cid]Entry

	// Re-computing this can get expensive so we memoize it.
	cached []Entry
}

// Entry is an entry in a want list, consisting of a cid and its priority
type Entry struct {
	Cid      cid.Cid
	Priority int32
	WantType pb.Message_Wantlist_WantType
}

// NewRefEntry creates a new reference tracked wantlist entry.
func NewRefEntry(c cid.Cid, p int32) Entry {
	return Entry{
		Cid:      c,
		Priority: p,
		WantType: pb.Message_Wantlist_Block,
	}
}

type entrySlice []Entry

func (es entrySlice) Len() int           { return len(es) }
func (es entrySlice) Swap(i, j int)      { es[i], es[j] = es[j], es[i] }
func (es entrySlice) Less(i, j int) bool { return es[i].Priority > es[j].Priority }

// New generates a new raw Wantlist
func New() *Wantlist {
	return &Wantlist{
		set: make(map[cid.Cid]Entry),
	}
}

// Len returns the number of entries in a wantlist.
func (w *Wantlist) Len() int {
	return len(w.set)
}

// Add adds an entry in a wantlist from CID & Priority, if not already present.
func (w *Wantlist) Add(c cid.Cid, priority int32, wantType pb.Message_Wantlist_WantType) bool {
	e, ok := w.set[c]

	// Adding want-have should not override want-block
	if ok && (e.WantType == pb.Message_Wantlist_Block || wantType == pb.Message_Wantlist_Have) {
		return false
	}

	w.put(c, Entry{
		Cid:      c,
		Priority: priority,
		WantType: wantType,
	})

	return true
}

// Remove removes the given cid from the wantlist.
func (w *Wantlist) Remove(c cid.Cid) bool {
	_, ok := w.set[c]
	if !ok {
		return false
	}

	w.delete(c)
	return true
}

// Remove removes the given cid from the wantlist, respecting the type:
// Remove with want-have will not remove an existing want-block.
func (w *Wantlist) RemoveType(c cid.Cid, wantType pb.Message_Wantlist_WantType) bool {
	e, ok := w.set[c]
	if !ok {
		return false
	}

	// Removing want-have should not remove want-block
	if e.WantType == pb.Message_Wantlist_Block && wantType == pb.Message_Wantlist_Have {
		return false
	}

	w.delete(c)
	return true
}

func (w *Wantlist) delete(c cid.Cid) {
	delete(w.set, c)
	w.cached = nil
}

func (w *Wantlist) put(c cid.Cid, e Entry) {
	w.cached = nil
	w.set[c] = e
}

// Contains returns the entry, if present, for the given CID, plus whether it
// was present.
func (w *Wantlist) Contains(c cid.Cid) (Entry, bool) {
	e, ok := w.set[c]
	return e, ok
}

// Entries returns all wantlist entries for a want list, sorted by priority.
//
// DO NOT MODIFY. The returned list is cached.
func (w *Wantlist) Entries() []Entry {
	if w.cached != nil {
		return w.cached
	}
	es := make([]Entry, 0, len(w.set))
	for _, e := range w.set {
		es = append(es, e)
	}
	sort.Sort(entrySlice(es))
	w.cached = es
	return es[0:len(es):len(es)]
}

// Absorb all the entries in other into this want list
func (w *Wantlist) Absorb(other *Wantlist) {
	// Invalidate the cache up-front to avoid doing any work trying to keep it up-to-date.
	w.cached = nil

	for _, e := range other.Entries() {
		w.Add(e.Cid, e.Priority, e.WantType)
	}
}
