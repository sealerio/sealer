package peermanager

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"

	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/notifications"
)

// PeerQueue is a process that sends messages to a peer
type PeerQueue interface {
	PeerProcess
	AllocateAndBuildMessage(blkSize uint64, buildMessageFn func(*gsmsg.Builder), notifees []notifications.Notifee)
}

// PeerQueueFactory provides a function that will create a PeerQueue.
type PeerQueueFactory func(ctx context.Context, p peer.ID) PeerQueue

// PeerMessageManager manages message queues for peers
type PeerMessageManager struct {
	*PeerManager
}

// NewMessageManager generates a new manger for sending messages
func NewMessageManager(ctx context.Context, createPeerQueue PeerQueueFactory) *PeerMessageManager {
	return &PeerMessageManager{
		PeerManager: New(ctx, func(ctx context.Context, p peer.ID) PeerHandler {
			return createPeerQueue(ctx, p)
		}),
	}
}

// BuildMessage allows you to modify the next message that is sent for the given peer
// If blkSize > 0, message building may block until enough memory has been freed from the queues to allocate the message.
func (pmm *PeerMessageManager) AllocateAndBuildMessage(p peer.ID, blkSize uint64, buildMessageFn func(*gsmsg.Builder), notifees []notifications.Notifee) {
	pq := pmm.GetProcess(p).(PeerQueue)
	pq.AllocateAndBuildMessage(blkSize, buildMessageFn, notifees)
}
