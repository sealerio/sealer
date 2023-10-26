package pubsub

import (
	"fmt"

	pb "github.com/libp2p/go-libp2p-pubsub/pb"

	"github.com/libp2p/go-libp2p-core/peer"
)

// NewMessageCache creates a sliding window cache that remembers messages for as
// long as `history` slots.
//
// When queried for messages to advertise, the cache only returns messages in
// the last `gossip` slots.
//
// The `gossip` parameter must be smaller or equal to `history`, or this
// function will panic.
//
// The slack between `gossip` and `history` accounts for the reaction time
// between when a message is advertised via IHAVE gossip, and the peer pulls it
// via an IWANT command.
func NewMessageCache(gossip, history int) *MessageCache {
	if gossip > history {
		err := fmt.Errorf("invalid parameters for message cache; gossip slots (%d) cannot be larger than history slots (%d)",
			gossip, history)
		panic(err)
	}
	return &MessageCache{
		msgs:    make(map[string]*pb.Message),
		peertx:  make(map[string]map[peer.ID]int),
		history: make([][]CacheEntry, history),
		gossip:  gossip,
		msgID:   DefaultMsgIdFn,
	}
}

type MessageCache struct {
	msgs    map[string]*pb.Message
	peertx  map[string]map[peer.ID]int
	history [][]CacheEntry
	gossip  int
	msgID   MsgIdFunction
}

func (mc *MessageCache) SetMsgIdFn(msgID MsgIdFunction) {
	mc.msgID = msgID
}

type CacheEntry struct {
	mid   string
	topic string
}

func (mc *MessageCache) Put(msg *pb.Message) {
	mid := mc.msgID(msg)
	mc.msgs[mid] = msg
	mc.history[0] = append(mc.history[0], CacheEntry{mid: mid, topic: msg.GetTopic()})
}

func (mc *MessageCache) Get(mid string) (*pb.Message, bool) {
	m, ok := mc.msgs[mid]
	return m, ok
}

func (mc *MessageCache) GetForPeer(mid string, p peer.ID) (*pb.Message, int, bool) {
	m, ok := mc.msgs[mid]
	if !ok {
		return nil, 0, false
	}

	tx, ok := mc.peertx[mid]
	if !ok {
		tx = make(map[peer.ID]int)
		mc.peertx[mid] = tx
	}
	tx[p]++

	return m, tx[p], true
}

func (mc *MessageCache) GetGossipIDs(topic string) []string {
	var mids []string
	for _, entries := range mc.history[:mc.gossip] {
		for _, entry := range entries {
			if entry.topic == topic {
				mids = append(mids, entry.mid)
			}
		}
	}
	return mids
}

func (mc *MessageCache) Shift() {
	last := mc.history[len(mc.history)-1]
	for _, entry := range last {
		delete(mc.msgs, entry.mid)
		delete(mc.peertx, entry.mid)
	}
	for i := len(mc.history) - 2; i >= 0; i-- {
		mc.history[i+1] = mc.history[i]
	}
	mc.history[0] = nil
}
