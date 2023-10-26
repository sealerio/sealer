package cidset

import (
	"errors"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/fluent"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"

	"github.com/ipfs/go-graphsync/ipldutil"
)

// EncodeCidSet encodes a cid set into bytes for the do-no-send-cids extension
func EncodeCidSet(cids *cid.Set) ([]byte, error) {
	list := fluent.MustBuildList(basicnode.Prototype.List, int64(cids.Len()), func(la fluent.ListAssembler) {
		_ = cids.ForEach(func(c cid.Cid) error {
			la.AssembleValue().AssignLink(cidlink.Link{Cid: c})
			return nil
		})
	})
	return ipldutil.EncodeNode(list)
}

// DecodeCidSet decode a cid set from data for the do-no-send-cids extension
func DecodeCidSet(data []byte) (*cid.Set, error) {
	list, err := ipldutil.DecodeNode(data)
	if err != nil {
		return nil, err
	}
	set := cid.NewSet()
	iter := list.ListIterator()
	for !iter.Done() {
		_, next, err := iter.Next()
		if err != nil {
			return nil, err
		}
		link, err := next.AsLink()
		if err != nil {
			return nil, err
		}
		asCidLink, ok := link.(cidlink.Link)
		if !ok {
			return nil, errors.New("contained non CID link")
		}
		set.Add(asCidLink.Cid)
	}
	return set, nil
}
