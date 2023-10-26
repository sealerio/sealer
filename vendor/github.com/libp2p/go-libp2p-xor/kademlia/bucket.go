package kademlia

import (
	"github.com/libp2p/go-libp2p-xor/key"
	"github.com/libp2p/go-libp2p-xor/trie"
)

// BucketAtDepth returns the bucket in the routing table at a given depth.
// A bucket at depth D holds contacts that share a prefix of exactly D bits with node.
func BucketAtDepth(node key.Key, table *trie.Trie, depth int) *trie.Trie {
	dir := node.BitAt(depth)
	if table.IsLeaf() {
		return nil
	} else {
		if depth == 0 {
			return table.Branch[1-dir]
		} else {
			return BucketAtDepth(node, table.Branch[dir], depth-1)
		}
	}
}

// ClosestN will return the count closest keys to the given key.
func ClosestN(node key.Key, table *trie.Trie, count int) []key.Key {
	return closestAtDepth(node, table, 0, count, make([]key.Key, 0, count))
}

func closestAtDepth(node key.Key, table *trie.Trie, depth int, count int, found []key.Key) []key.Key {
	// If we've already found enough peers, abort.
	if count == len(found) {
		return found
	}

	// Find the closest direction.
	dir := node.BitAt(depth)
	var chosenDir byte
	if table.Branch[dir] != nil {
		// There are peers in the "closer" direction.
		chosenDir = dir
	} else if table.Branch[1-dir] != nil {
		// There are peers in the "less closer" direction.
		chosenDir = 1 - dir
	} else if table.Key != nil {
		// We've found a leaf
		return append(found, table.Key)
	} else {
		// We've found an empty node?
		return found
	}

	// Add peers from the closest direction first, then from the other direction.
	found = closestAtDepth(node, table.Branch[chosenDir], depth+1, count, found)
	found = closestAtDepth(node, table.Branch[1-chosenDir], depth+1, count, found)
	return found
}
