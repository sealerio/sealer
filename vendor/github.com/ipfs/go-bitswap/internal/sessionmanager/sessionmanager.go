package sessionmanager

import (
	"context"
	"sync"
	"time"

	cid "github.com/ipfs/go-cid"
	delay "github.com/ipfs/go-ipfs-delay"

	bsbpm "github.com/ipfs/go-bitswap/internal/blockpresencemanager"
	notifications "github.com/ipfs/go-bitswap/internal/notifications"
	bssession "github.com/ipfs/go-bitswap/internal/session"
	bssim "github.com/ipfs/go-bitswap/internal/sessioninterestmanager"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// Session is a session that is managed by the session manager
type Session interface {
	exchange.Fetcher
	ID() uint64
	ReceiveFrom(peer.ID, []cid.Cid, []cid.Cid, []cid.Cid)
	Shutdown()
}

// SessionFactory generates a new session for the SessionManager to track.
type SessionFactory func(
	ctx context.Context,
	sm bssession.SessionManager,
	id uint64,
	sprm bssession.SessionPeerManager,
	sim *bssim.SessionInterestManager,
	pm bssession.PeerManager,
	bpm *bsbpm.BlockPresenceManager,
	notif notifications.PubSub,
	provSearchDelay time.Duration,
	rebroadcastDelay delay.D,
	self peer.ID) Session

// PeerManagerFactory generates a new peer manager for a session.
type PeerManagerFactory func(ctx context.Context, id uint64) bssession.SessionPeerManager

// SessionManager is responsible for creating, managing, and dispatching to
// sessions.
type SessionManager struct {
	ctx                    context.Context
	sessionFactory         SessionFactory
	sessionInterestManager *bssim.SessionInterestManager
	peerManagerFactory     PeerManagerFactory
	blockPresenceManager   *bsbpm.BlockPresenceManager
	peerManager            bssession.PeerManager
	notif                  notifications.PubSub

	// Sessions
	sessLk   sync.RWMutex
	sessions map[uint64]Session

	// Session Index
	sessIDLk sync.Mutex
	sessID   uint64

	self peer.ID
}

// New creates a new SessionManager.
func New(ctx context.Context, sessionFactory SessionFactory, sessionInterestManager *bssim.SessionInterestManager, peerManagerFactory PeerManagerFactory,
	blockPresenceManager *bsbpm.BlockPresenceManager, peerManager bssession.PeerManager, notif notifications.PubSub, self peer.ID) *SessionManager {

	return &SessionManager{
		ctx:                    ctx,
		sessionFactory:         sessionFactory,
		sessionInterestManager: sessionInterestManager,
		peerManagerFactory:     peerManagerFactory,
		blockPresenceManager:   blockPresenceManager,
		peerManager:            peerManager,
		notif:                  notif,
		sessions:               make(map[uint64]Session),
		self:                   self,
	}
}

// NewSession initializes a session with the given context, and adds to the
// session manager.
func (sm *SessionManager) NewSession(ctx context.Context,
	provSearchDelay time.Duration,
	rebroadcastDelay delay.D) exchange.Fetcher {
	id := sm.GetNextSessionID()

	pm := sm.peerManagerFactory(ctx, id)
	session := sm.sessionFactory(ctx, sm, id, pm, sm.sessionInterestManager, sm.peerManager, sm.blockPresenceManager, sm.notif, provSearchDelay, rebroadcastDelay, sm.self)

	sm.sessLk.Lock()
	if sm.sessions != nil { // check if SessionManager was shutdown
		sm.sessions[id] = session
	}
	sm.sessLk.Unlock()

	return session
}

func (sm *SessionManager) Shutdown() {
	sm.sessLk.Lock()

	sessions := make([]Session, 0, len(sm.sessions))
	for _, ses := range sm.sessions {
		sessions = append(sessions, ses)
	}

	// Ensure that if Shutdown() is called twice we only shut down
	// the sessions once
	sm.sessions = nil

	sm.sessLk.Unlock()

	for _, ses := range sessions {
		ses.Shutdown()
	}
}

func (sm *SessionManager) RemoveSession(sesid uint64) {
	// Remove session from SessionInterestManager - returns the keys that no
	// session is interested in anymore.
	cancelKs := sm.sessionInterestManager.RemoveSession(sesid)

	// Cancel keys that no session is interested in anymore
	sm.cancelWants(cancelKs)

	sm.sessLk.Lock()
	defer sm.sessLk.Unlock()

	// Clean up session
	if sm.sessions != nil { // check if SessionManager was shutdown
		delete(sm.sessions, sesid)
	}
}

// GetNextSessionID returns the next sequential identifier for a session.
func (sm *SessionManager) GetNextSessionID() uint64 {
	sm.sessIDLk.Lock()
	defer sm.sessIDLk.Unlock()

	sm.sessID++
	return sm.sessID
}

// ReceiveFrom is called when a new message is received
func (sm *SessionManager) ReceiveFrom(ctx context.Context, p peer.ID, blks []cid.Cid, haves []cid.Cid, dontHaves []cid.Cid) {
	// Record block presence for HAVE / DONT_HAVE
	sm.blockPresenceManager.ReceiveFrom(p, haves, dontHaves)

	// Notify each session that is interested in the blocks / HAVEs / DONT_HAVEs
	for _, id := range sm.sessionInterestManager.InterestedSessions(blks, haves, dontHaves) {
		sm.sessLk.RLock()
		if sm.sessions == nil { // check if SessionManager was shutdown
			sm.sessLk.RUnlock()
			return
		}
		sess, ok := sm.sessions[id]
		sm.sessLk.RUnlock()

		if ok {
			sess.ReceiveFrom(p, blks, haves, dontHaves)
		}
	}

	// Send CANCEL to all peers with want-have / want-block
	sm.peerManager.SendCancels(ctx, blks)
}

// CancelSessionWants is called when a session cancels wants because a call to
// GetBlocks() is cancelled
func (sm *SessionManager) CancelSessionWants(sesid uint64, wants []cid.Cid) {
	// Remove session's interest in the given blocks - returns the keys that no
	// session is interested in anymore.
	cancelKs := sm.sessionInterestManager.RemoveSessionInterested(sesid, wants)
	sm.cancelWants(cancelKs)
}

func (sm *SessionManager) cancelWants(wants []cid.Cid) {
	// Free up block presence tracking for keys that no session is interested
	// in anymore
	sm.blockPresenceManager.RemoveKeys(wants)

	// Send CANCEL to all peers for blocks that no session is interested in
	// anymore.
	// Note: use bitswap context because session context may already be Done.
	sm.peerManager.SendCancels(sm.ctx, wants)
}
