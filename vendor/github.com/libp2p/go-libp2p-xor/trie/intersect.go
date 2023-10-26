package trie

import (
	"github.com/libp2p/go-libp2p-xor/key"
)

func IntersectKeySlices(p, q []key.Key) []key.Key {
	hat := []key.Key{}
	for _, p := range p {
		if keyIsIn(p, q) && !keyIsIn(p, hat) {
			hat = append(hat, p)
		}
	}
	return hat
}

func keyIsIn(q key.Key, s []key.Key) bool {
	for _, s := range s {
		if key.Equal(q, s) {
			return true
		}
	}
	return false
}

// Intersect computes the intersection of the keys in p and q.
// p and q must be non-nil. The returned trie is never nil.
func Intersect(p, q *Trie) *Trie {
	return IntersectAtDepth(0, p, q)
}

func IntersectAtDepth(depth int, p, q *Trie) *Trie {
	switch {
	case p.IsLeaf() && q.IsLeaf():
		if p.IsEmpty() || q.IsEmpty() {
			return &Trie{} // empty set
		} else {
			if key.Equal(p.Key, q.Key) {
				return &Trie{Key: p.Key} // singleton
			} else {
				return &Trie{} // empty set
			}
		}
	case p.IsLeaf() && !q.IsLeaf():
		if p.IsEmpty() {
			return &Trie{} // empty set
		} else {
			if _, found := q.FindAtDepth(depth, p.Key); found {
				return &Trie{Key: p.Key}
			} else {
				return &Trie{} // empty set
			}
		}
	case !p.IsLeaf() && q.IsLeaf():
		return IntersectAtDepth(depth, q, p)
	case !p.IsLeaf() && !q.IsLeaf():
		disjointUnion := &Trie{
			Branch: [2]*Trie{
				IntersectAtDepth(depth+1, p.Branch[0], q.Branch[0]),
				IntersectAtDepth(depth+1, p.Branch[1], q.Branch[1]),
			},
		}
		disjointUnion.shrink()
		return disjointUnion
	}
	panic("unreachable")
}
