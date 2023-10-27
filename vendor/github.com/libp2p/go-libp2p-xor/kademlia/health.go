package kademlia

import (
	"encoding/json"
	"sort"

	"github.com/libp2p/go-libp2p-xor/key"
	"github.com/libp2p/go-libp2p-xor/trie"
)

// TableHealthReport describes the discrepancy between a node's routing table from the theoretical ideal,
// given knowledge of all nodes present in the network.
// TODO: Make printable in a way easy to ingest in Python/matplotlib for viewing in a Jupyter notebook.
// E.g. one would like to see a histogram of (IdealDepth - ActualDepth) across all tables.
type TableHealthReport struct {
	// IdealDepth is the depth that the node's routing table should have.
	IdealDepth int
	// ActualDepth is the depth that the node's routing table has.
	ActualDepth int
	// Bucket contains the individual health reports for each of the node's routing buckets.
	Bucket []*BucketHealthReport
}

func (th *TableHealthReport) String() string {
	b, _ := json.Marshal(th)
	return string(b)
}

// BucketHealth describes the discrepancy between a node's routing bucket and the theoretical ideal,
// given knowledge of all nodes present in the network (aka the "known" nodes).
type BucketHealthReport struct {
	// Depth is the bucket depth, starting from zero.
	Depth int
	// MaxKnownContacts is the number of all known network nodes,
	// which are eligible to be in this bucket.
	MaxKnownContacts int
	// ActualKnownContacts is the number of known network nodes,
	// that are actually in the node's routing table.
	ActualKnownContacts int
	// ActualUnknownContacts is the number of contacts in the node's routing table,
	// that are not known to be in the network currently.
	ActualUnknownContacts int
}

func (bh *BucketHealthReport) String() string {
	b, _ := json.Marshal(bh)
	return string(b)
}

// sortedBucketHealthReport sorts bucket health reports in ascending order of depth.
type sortedBucketHealthReport []*BucketHealthReport

func (s sortedBucketHealthReport) Less(i, j int) bool {
	return s[i].Depth < s[j].Depth
}

func (s sortedBucketHealthReport) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortedBucketHealthReport) Len() int {
	return len(s)
}

type Table struct {
	Node     key.Key
	Contacts []key.Key
}

// AllTablesHealth computes health reports for a network of nodes, whose routing contacts are given.
func AllTablesHealth(tables []*Table) (report []*TableHealthReport) {
	// Construct global network view trie
	knownNodes := trie.New()
	for _, table := range tables {
		knownNodes.Add(table.Node)
	}
	// Compute individual table health
	for _, table := range tables {
		report = append(report, TableHealth(table.Node, table.Contacts, knownNodes))
	}
	return
}

func TableHealthFromSets(node key.Key, nodeContacts []key.Key, knownNodes []key.Key) *TableHealthReport {
	knownNodesTrie := trie.New()
	for _, k := range knownNodes {
		knownNodesTrie.Add(k)
	}
	return TableHealth(node, nodeContacts, knownNodesTrie)
}

// TableHealth computes the health report for a node,
// given its routing contacts and a list of all known nodes in the network currently.
func TableHealth(node key.Key, nodeContacts []key.Key, knownNodes *trie.Trie) *TableHealthReport {
	// Reconstruct the node's routing table as a trie
	nodeTable := trie.New()
	nodeTable.Add(node)
	for _, u := range nodeContacts {
		nodeTable.Add(u)
	}
	// Compute health report
	idealDepth, _ := knownNodes.Find(node)
	actualDepth, _ := nodeTable.Find(node)
	return &TableHealthReport{
		IdealDepth:  idealDepth,
		ActualDepth: actualDepth,
		Bucket:      BucketHealth(node, nodeTable, knownNodes),
	}
}

// BucketHealth computes the health report for each bucket in a node's routing table,
// given the node's routing table and a list of all known nodes in the network currently.
func BucketHealth(node key.Key, nodeTable, knownNodes *trie.Trie) []*BucketHealthReport {
	r := walkBucketHealth(0, node, nodeTable, knownNodes)
	sort.Sort(sortedBucketHealthReport(r))
	return r
}

func walkBucketHealth(depth int, node key.Key, nodeTable, knownNodes *trie.Trie) []*BucketHealthReport {
	if nodeTable.IsLeaf() {
		return nil
	} else {
		dir := node.BitAt(depth)
		switch {
		//
		case knownNodes == nil || knownNodes.IsEmptyLeaf():
			r := walkBucketHealth(depth+1, node, nodeTable.Branch[dir], nil)
			return append(r,
				&BucketHealthReport{
					Depth:                 depth,
					MaxKnownContacts:      0,
					ActualKnownContacts:   0,
					ActualUnknownContacts: nodeTable.Branch[1-dir].Size(),
				})
		case knownNodes.IsNonEmptyLeaf():
			if knownNodes.Key.BitAt(depth) == dir {
				r := walkBucketHealth(depth+1, node, nodeTable.Branch[dir], knownNodes)
				return append(r,
					&BucketHealthReport{
						Depth:                 depth,
						MaxKnownContacts:      0,
						ActualKnownContacts:   0,
						ActualUnknownContacts: nodeTable.Branch[1-dir].Size(),
					})
			} else {
				r := walkBucketHealth(depth+1, node, nodeTable.Branch[dir], nil)
				return append(r, bucketReportFromTries(depth+1, nodeTable.Branch[1-dir], knownNodes))
			}
		case !knownNodes.IsLeaf():
			r := walkBucketHealth(depth+1, node, nodeTable.Branch[dir], knownNodes.Branch[dir])
			return append(r,
				bucketReportFromTries(depth+1, nodeTable.Branch[1-dir], knownNodes.Branch[1-dir]))
		default:
			panic("unreachable")
		}
	}
}

func bucketReportFromTries(depth int, actualBucket, maxBucket *trie.Trie) *BucketHealthReport {
	actualKnown := trie.IntersectAtDepth(depth, actualBucket, maxBucket)
	actualKnownSize := actualKnown.Size()
	return &BucketHealthReport{
		Depth:                 depth,
		MaxKnownContacts:      maxBucket.Size(),
		ActualKnownContacts:   actualKnownSize,
		ActualUnknownContacts: actualBucket.Size() - actualKnownSize,
	}
}
