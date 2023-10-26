package trie

import (
	"github.com/libp2p/go-libp2p-xor/key"
)

type InvariantDiscrepancy struct {
	Reason            string
	PathToDiscrepancy string
	KeyAtDiscrepancy  string
}

// CheckInvariant panics of the trie does not meet its invariant.
func (trie *Trie) CheckInvariant() *InvariantDiscrepancy {
	return trie.checkInvariant(0, nil)
}

func (trie *Trie) checkInvariant(depth int, pathSoFar *triePath) *InvariantDiscrepancy {
	switch {
	case trie.IsEmptyLeaf():
		return nil
	case trie.IsNonEmptyLeaf():
		if !pathSoFar.matchesKey(trie.Key) {
			return &InvariantDiscrepancy{
				Reason:            "key found at invalid location in trie",
				PathToDiscrepancy: pathSoFar.BitString(),
				KeyAtDiscrepancy:  trie.Key.BitString(),
			}
		}
		return nil
	default:
		if trie.IsEmpty() {
			b0, b1 := trie.Branch[0], trie.Branch[1]
			if d0 := b0.checkInvariant(depth+1, pathSoFar.Push(0)); d0 != nil {
				return d0
			}
			if d1 := b1.checkInvariant(depth+1, pathSoFar.Push(1)); d1 != nil {
				return d1
			}
			switch {
			case b0.IsEmptyLeaf() && b1.IsEmptyLeaf():
				return &InvariantDiscrepancy{
					Reason:            "intermediate node with two empty leaves",
					PathToDiscrepancy: pathSoFar.BitString(),
					KeyAtDiscrepancy:  "none",
				}
			case b0.IsEmptyLeaf() && b1.IsNonEmptyLeaf():
				return &InvariantDiscrepancy{
					Reason:            "intermediate node with one empty leaf/0",
					PathToDiscrepancy: pathSoFar.BitString(),
					KeyAtDiscrepancy:  "none",
				}
			case b0.IsNonEmptyLeaf() && b1.IsEmptyLeaf():
				return &InvariantDiscrepancy{
					Reason:            "intermediate node with one empty leaf/1",
					PathToDiscrepancy: pathSoFar.BitString(),
					KeyAtDiscrepancy:  "none",
				}
			default:
				return nil
			}
		} else {
			return &InvariantDiscrepancy{
				Reason:            "intermediate node with a key",
				PathToDiscrepancy: pathSoFar.BitString(),
				KeyAtDiscrepancy:  trie.Key.BitString(),
			}
		}
	}
}

type triePath struct {
	parent *triePath
	bit    byte
}

func (p *triePath) Push(bit byte) *triePath {
	return &triePath{parent: p, bit: bit}
}

func (p *triePath) RootPath() []byte {
	if p == nil {
		return nil
	} else {
		return append(p.parent.RootPath(), p.bit)
	}
}

func (p *triePath) matchesKey(k key.Key) bool {
	// Slower, but more explicit:
	// for i, b := range p.RootPath() {
	// 	if k.BitAt(i) != b {
	// 		return false
	// 	}
	// }
	// return true
	ok, _ := p.walk(k, 0)
	return ok
}

func (p *triePath) walk(k key.Key, depthToLeaf int) (ok bool, depthToRoot int) {
	if p == nil {
		return true, 0
	} else {
		parOk, parDepthToRoot := p.parent.walk(k, depthToLeaf+1)
		return k.BitAt(parDepthToRoot) == p.bit && parOk, parDepthToRoot + 1
	}
}

func (p *triePath) BitString() string {
	return p.bitString(0)
}

func (p *triePath) bitString(depthToLeaf int) string {
	if p == nil {
		return ""
	} else {
		switch {
		case p.bit == 0:
			return p.parent.bitString(depthToLeaf+1) + "0"
		case p.bit == 1:
			return p.parent.bitString(depthToLeaf+1) + "1"
		default:
			panic("bit digit > 1")
		}
	}
}
