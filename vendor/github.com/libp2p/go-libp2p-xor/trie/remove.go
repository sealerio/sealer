package trie

import (
	"github.com/libp2p/go-libp2p-xor/key"
)

// Remove removes the key q from the trie. Remove mutates the trie.
// TODO: Also implement an immutable version of Remove.
func (trie *Trie) Remove(q key.Key) (removedDepth int, removed bool) {
	return trie.RemoveAtDepth(0, q)
}

func (trie *Trie) RemoveAtDepth(depth int, q key.Key) (reachedDepth int, removed bool) {
	switch {
	case trie.IsEmptyLeaf():
		return depth, false
	case trie.IsNonEmptyLeaf():
		trie.Key = nil
		return depth, true
	default:
		if d, removed := trie.Branch[q.BitAt(depth)].RemoveAtDepth(depth+1, q); removed {
			trie.shrink()
			return d, true
		} else {
			return d, false
		}
	}
}

func Remove(trie *Trie, q key.Key) *Trie {
	return RemoveAtDepth(0, trie, q)
}

func RemoveAtDepth(depth int, trie *Trie, q key.Key) *Trie {
	switch {
	case trie.IsEmptyLeaf():
		return trie
	case trie.IsNonEmptyLeaf() && !key.Equal(trie.Key, q):
		return trie
	case trie.IsNonEmptyLeaf() && key.Equal(trie.Key, q):
		return &Trie{}
	default:
		dir := q.BitAt(depth)
		afterDelete := RemoveAtDepth(depth+1, trie.Branch[dir], q)
		if afterDelete == trie.Branch[dir] {
			return trie
		}
		copy := &Trie{}
		copy.Branch[dir] = afterDelete
		copy.Branch[1-dir] = trie.Branch[1-dir]
		copy.shrink()
		return copy
	}
}
