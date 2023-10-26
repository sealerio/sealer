package ipldgit

import (
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	mh "github.com/multiformats/go-multihash"
)

func shaToCid(sha []byte) cid.Cid {
	h, _ := mh.Encode(sha, mh.SHA1)
	return cid.NewCidV1(cid.GitRaw, h)
}

func cidToSha(c cid.Cid) []byte {
	h := c.Hash()
	return h[len(h)-20:]
}

func sha(l ipld.Link) []byte {
	cl, ok := l.(cidlink.Link)
	if !ok {
		return nil
	}
	return cidToSha(cl.Cid)
}

func (l Link) sha() []byte {
	cl, ok := l.x.(cidlink.Link)
	if !ok {
		return nil
	}
	return cidToSha(cl.Cid)
}

func (l Tree_Link) sha() []byte {
	cl, ok := l.x.(cidlink.Link)
	if !ok {
		return nil
	}
	return cidToSha(cl.Cid)
}

func (l Commit_Link) sha() []byte {
	cl, ok := l.x.(cidlink.Link)
	if !ok {
		return nil
	}
	return cidToSha(cl.Cid)
}
