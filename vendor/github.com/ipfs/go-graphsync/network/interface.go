package network

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	gsmsg "github.com/ipfs/go-graphsync/message"
)

var (
	// ProtocolGraphsync is the protocol identifier for graphsync messages
	ProtocolGraphsync protocol.ID = "/ipfs/graphsync/1.0.0"
)

// GraphSyncNetwork provides network connectivity for GraphSync.
type GraphSyncNetwork interface {

	// SendMessage sends a GraphSync message to a peer.
	SendMessage(
		context.Context,
		peer.ID,
		gsmsg.GraphSyncMessage) error

	// SetDelegate registers the Receiver to handle messages received from the
	// network.
	SetDelegate(Receiver)

	// ConnectTo establishes a connection to the given peer
	ConnectTo(context.Context, peer.ID) error

	NewMessageSender(context.Context, peer.ID, MessageSenderOpts) (MessageSender, error)

	ConnectionManager() ConnManager
}

// MessageSenderOpts sets parameters for a message sender
type MessageSenderOpts struct {
	SendTimeout time.Duration
}

// ConnManager provides the methods needed to protect and unprotect connections
type ConnManager interface {
	Protect(peer.ID, string)
	Unprotect(peer.ID, string) bool
}

// MessageSender is an interface to send messages to a peer
type MessageSender interface {
	SendMsg(context.Context, gsmsg.GraphSyncMessage) error
	Close() error
	Reset() error
}

// Receiver is an interface for receiving messages from the GraphSyncNetwork.
type Receiver interface {
	ReceiveMessage(
		ctx context.Context,
		sender peer.ID,
		incoming gsmsg.GraphSyncMessage)

	ReceiveError(p peer.ID, err error)

	Connected(p peer.ID)
	Disconnected(p peer.ID)
}
