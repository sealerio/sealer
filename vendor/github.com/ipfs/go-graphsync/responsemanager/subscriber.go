package responsemanager

import (
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/messagequeue"
	"github.com/ipfs/go-graphsync/network"
	"github.com/ipfs/go-graphsync/notifications"
)

// RequestCloser can cancel request on a network error
type RequestCloser interface {
	CloseWithNetworkError(p peer.ID, requestID graphsync.RequestID)
}

type subscriber struct {
	p                     peer.ID
	request               gsmsg.GraphSyncRequest
	requestCloser         RequestCloser
	blockSentListeners    BlockSentListeners
	networkErrorListeners NetworkErrorListeners
	completedListeners    CompletedListeners
	connManager           network.ConnManager
}

func (s *subscriber) OnNext(topic notifications.Topic, event notifications.Event) {
	responseEvent, ok := event.(messagequeue.Event)
	if !ok {
		return
	}
	blockData, isBlockData := topic.(graphsync.BlockData)
	if isBlockData {
		switch responseEvent.Name {
		case messagequeue.Error:
			s.networkErrorListeners.NotifyNetworkErrorListeners(s.p, s.request, responseEvent.Err)
			s.requestCloser.CloseWithNetworkError(s.p, s.request.ID())
		case messagequeue.Sent:
			s.blockSentListeners.NotifyBlockSentListeners(s.p, s.request, blockData)
		}
		return
	}
	status, isStatus := topic.(graphsync.ResponseStatusCode)
	if isStatus {
		s.connManager.Unprotect(s.p, s.request.ID().Tag())
		switch responseEvent.Name {
		case messagequeue.Error:
			s.networkErrorListeners.NotifyNetworkErrorListeners(s.p, s.request, responseEvent.Err)
			s.requestCloser.CloseWithNetworkError(s.p, s.request.ID())
		case messagequeue.Sent:
			s.completedListeners.NotifyCompletedListeners(s.p, s.request, status)
		}
	}
}

func (s *subscriber) OnClose(topic notifications.Topic) {

}
