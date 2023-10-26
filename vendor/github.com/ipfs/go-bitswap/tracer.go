package bitswap

import (
	bsmsg "github.com/ipfs/go-bitswap/message"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// Tracer provides methods to access all messages sent and received by Bitswap.
// This interface can be used to implement various statistics (this is original intent).
type Tracer interface {
	MessageReceived(peer.ID, bsmsg.BitSwapMessage)
	MessageSent(peer.ID, bsmsg.BitSwapMessage)
}

// Configures Bitswap to use given tracer.
func WithTracer(tap Tracer) Option {
	return func(bs *Bitswap) {
		bs.tracer = tap
	}
}
