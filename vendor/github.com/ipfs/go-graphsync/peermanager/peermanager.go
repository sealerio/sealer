package peermanager

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
)

// PeerProcess is any process that provides services for a peer
type PeerProcess interface {
	Startup()
	Shutdown()
}

type PeerHandler interface{}

// PeerProcessFactory provides a function that will create a PeerQueue.
type PeerProcessFactory func(ctx context.Context, p peer.ID) PeerHandler

type peerProcessInstance struct {
	refcnt  int
	process PeerHandler
}

// PeerManager manages a pool of peers and sends messages to peers in the pool.
type PeerManager struct {
	peerProcesses   map[peer.ID]*peerProcessInstance
	peerProcessesLk sync.RWMutex

	createPeerProcess PeerProcessFactory
	ctx               context.Context
}

// New creates a new PeerManager, given a context and a peerQueueFactory.
func New(ctx context.Context, createPeerQueue PeerProcessFactory) *PeerManager {
	return &PeerManager{
		peerProcesses:     make(map[peer.ID]*peerProcessInstance),
		createPeerProcess: createPeerQueue,
		ctx:               ctx,
	}
}

// ConnectedPeers returns a list of peers this PeerManager is managing.
func (pm *PeerManager) ConnectedPeers() []peer.ID {
	pm.peerProcessesLk.RLock()
	defer pm.peerProcessesLk.RUnlock()
	peers := make([]peer.ID, 0, len(pm.peerProcesses))
	for p := range pm.peerProcesses {
		peers = append(peers, p)
	}
	return peers
}

// Connected is called to add a new peer to the pool
func (pm *PeerManager) Connected(p peer.ID) {
	pm.peerProcessesLk.Lock()
	pq := pm.getOrCreate(p)
	pq.refcnt++
	pm.peerProcessesLk.Unlock()
}

// Disconnected is called to remove a peer from the pool.
func (pm *PeerManager) Disconnected(p peer.ID) {
	pm.peerProcessesLk.Lock()
	pq, ok := pm.peerProcesses[p]
	if !ok {
		pm.peerProcessesLk.Unlock()
		return
	}

	pq.refcnt--
	if pq.refcnt > 0 {
		pm.peerProcessesLk.Unlock()
		return
	}

	delete(pm.peerProcesses, p)
	pm.peerProcessesLk.Unlock()

	if pprocess, ok := pq.process.(PeerProcess); ok {
		pprocess.Shutdown()
	}
}

// GetProcess returns the process for the given peer
func (pm *PeerManager) GetProcess(
	p peer.ID) PeerHandler {
	// Usually this this is just a read
	pm.peerProcessesLk.RLock()
	pqi, ok := pm.peerProcesses[p]
	if ok {
		pm.peerProcessesLk.RUnlock()
		return pqi.process
	}
	pm.peerProcessesLk.RUnlock()
	// but sometimes it involves a create (we still need to do get or create cause it's possible
	// another writer grabbed the Lock first and made the process)
	pm.peerProcessesLk.Lock()
	pqi = pm.getOrCreate(p)
	pm.peerProcessesLk.Unlock()
	return pqi.process
}

func (pm *PeerManager) getOrCreate(p peer.ID) *peerProcessInstance {
	pqi, ok := pm.peerProcesses[p]
	if !ok {
		pq := pm.createPeerProcess(pm.ctx, p)
		if pprocess, ok := pq.(PeerProcess); ok {
			pprocess.Startup()
		}
		pqi = &peerProcessInstance{0, pq}
		pm.peerProcesses[p] = pqi
	}
	return pqi
}
