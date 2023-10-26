package network

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
)

type ConnectionListener interface {
	PeerConnected(peer.ID)
	PeerDisconnected(peer.ID)
}

type connectEventManager struct {
	connListener ConnectionListener
	lk           sync.RWMutex
	conns        map[peer.ID]*connState
}

type connState struct {
	refs       int
	responsive bool
}

func newConnectEventManager(connListener ConnectionListener) *connectEventManager {
	return &connectEventManager{
		connListener: connListener,
		conns:        make(map[peer.ID]*connState),
	}
}

func (c *connectEventManager) Connected(p peer.ID) {
	c.lk.Lock()
	defer c.lk.Unlock()

	state, ok := c.conns[p]
	if !ok {
		state = &connState{responsive: true}
		c.conns[p] = state
	}
	state.refs++

	if state.refs == 1 && state.responsive {
		c.connListener.PeerConnected(p)
	}
}

func (c *connectEventManager) Disconnected(p peer.ID) {
	c.lk.Lock()
	defer c.lk.Unlock()

	state, ok := c.conns[p]
	if !ok {
		// Should never happen
		return
	}
	state.refs--

	if state.refs == 0 {
		if state.responsive {
			c.connListener.PeerDisconnected(p)
		}
		delete(c.conns, p)
	}
}

func (c *connectEventManager) MarkUnresponsive(p peer.ID) {
	c.lk.Lock()
	defer c.lk.Unlock()

	state, ok := c.conns[p]
	if !ok || !state.responsive {
		return
	}
	state.responsive = false

	c.connListener.PeerDisconnected(p)
}

func (c *connectEventManager) OnMessage(p peer.ID) {
	// This is a frequent operation so to avoid different message arrivals
	// getting blocked by a write lock, first take a read lock to check if
	// we need to modify state
	c.lk.RLock()
	state, ok := c.conns[p]
	responsive := ok && state.responsive
	c.lk.RUnlock()

	if !ok || responsive {
		return
	}

	// We need to make a modification so now take a write lock
	c.lk.Lock()
	defer c.lk.Unlock()

	// Note: state may have changed in the time between when read lock
	// was released and write lock taken, so check again
	state, ok = c.conns[p]
	if !ok || state.responsive {
		return
	}

	state.responsive = true
	c.connListener.PeerConnected(p)
}
