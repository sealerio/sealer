package bitswap

import (
	"sort"

	cid "github.com/ipfs/go-cid"
)

// Stat is a struct that provides various statistics on bitswap operations
type Stat struct {
	ProvideBufLen    int
	Wantlist         []cid.Cid
	Peers            []string
	BlocksReceived   uint64
	DataReceived     uint64
	BlocksSent       uint64
	DataSent         uint64
	DupBlksReceived  uint64
	DupDataReceived  uint64
	MessagesReceived uint64
}

// Stat returns aggregated statistics about bitswap operations
func (bs *Bitswap) Stat() (*Stat, error) {
	st := new(Stat)
	st.ProvideBufLen = len(bs.newBlocks)
	st.Wantlist = bs.GetWantlist()
	bs.counterLk.Lock()
	c := bs.counters
	st.BlocksReceived = c.blocksRecvd
	st.DupBlksReceived = c.dupBlocksRecvd
	st.DupDataReceived = c.dupDataRecvd
	st.BlocksSent = c.blocksSent
	st.DataSent = c.dataSent
	st.DataReceived = c.dataRecvd
	st.MessagesReceived = c.messagesRecvd
	bs.counterLk.Unlock()

	peers := bs.engine.Peers()
	st.Peers = make([]string, 0, len(peers))

	for _, p := range peers {
		st.Peers = append(st.Peers, p.Pretty())
	}
	sort.Strings(st.Peers)

	return st, nil
}
