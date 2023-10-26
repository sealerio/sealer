package trie

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p-xor/key"
)

// Trie is a trie for equal-length bit vectors, which stores values only in the leaves.
// Trie node invariants:
// (1) Either both branches are nil, or both are non-nil.
// (2) If branches are non-nil, key must be nil.
// (3) If both branches are leaves, then they are both non-empty (have keys).
type Trie struct {
	Branch [2]*Trie
	Key    key.Key
}

func New() *Trie {
	return &Trie{}
}

func FromKeys(k []key.Key) *Trie {
	t := New()
	for _, k := range k {
		t.Add(k)
	}
	return t
}

func FromKeysAtDepth(depth int, k []key.Key) *Trie {
	t := New()
	for _, k := range k {
		t.AddAtDepth(depth, k)
	}
	return t
}

func (trie *Trie) String() string {
	b, _ := json.Marshal(trie)
	return string(b)
}

func (trie *Trie) Depth() int {
	return trie.DepthAtDepth(0)
}

func (trie *Trie) DepthAtDepth(depth int) int {
	if trie.Branch[0] == nil && trie.Branch[1] == nil {
		return depth
	} else {
		return max(trie.Branch[0].DepthAtDepth(depth+1), trie.Branch[1].DepthAtDepth(depth+1))
	}
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// Size returns the number of keys added to the trie.
// In other words, it returns the number of non-empty leaves in the trie.
func (trie *Trie) Size() int {
	return trie.SizeAtDepth(0)
}

func (trie *Trie) SizeAtDepth(depth int) int {
	if trie.Branch[0] == nil && trie.Branch[1] == nil {
		if trie.IsEmpty() {
			return 0
		} else {
			return 1
		}
	} else {
		return trie.Branch[0].SizeAtDepth(depth+1) + trie.Branch[1].SizeAtDepth(depth+1)
	}
}

func (trie *Trie) IsEmpty() bool {
	return trie.Key == nil
}

func (trie *Trie) IsLeaf() bool {
	return trie.Branch[0] == nil && trie.Branch[1] == nil
}

func (trie *Trie) IsEmptyLeaf() bool {
	return trie.IsEmpty() && trie.IsLeaf()
}

func (trie *Trie) IsNonEmptyLeaf() bool {
	return !trie.IsEmpty() && trie.IsLeaf()
}

func (trie *Trie) shrink() {
	b0, b1 := trie.Branch[0], trie.Branch[1]
	switch {
	case b0.IsEmptyLeaf() && b1.IsEmptyLeaf():
		trie.Branch[0], trie.Branch[1] = nil, nil
	case b0.IsEmptyLeaf() && b1.IsNonEmptyLeaf():
		trie.Key = b1.Key
		trie.Branch[0], trie.Branch[1] = nil, nil
	case b0.IsNonEmptyLeaf() && b1.IsEmptyLeaf():
		trie.Key = b0.Key
		trie.Branch[0], trie.Branch[1] = nil, nil
	}
}
