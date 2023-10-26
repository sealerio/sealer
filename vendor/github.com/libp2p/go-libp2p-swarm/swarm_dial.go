package swarm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/transport"

	addrutil "github.com/libp2p/go-addr-util"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

// Diagram of dial sync:
//
//   many callers of Dial()   synched w.  dials many addrs       results to callers
//  ----------------------\    dialsync    use earliest            /--------------
//  -----------------------\              |----------\           /----------------
//  ------------------------>------------<-------     >---------<-----------------
//  -----------------------|              \----x                 \----------------
//  ----------------------|                \-----x                \---------------
//                                         any may fail          if no addr at end
//                                                             retry dialAttempt x

var (
	// ErrDialBackoff is returned by the backoff code when a given peer has
	// been dialed too frequently
	ErrDialBackoff = errors.New("dial backoff")

	// ErrDialToSelf is returned if we attempt to dial our own peer
	ErrDialToSelf = errors.New("dial to self attempted")

	// ErrNoTransport is returned when we don't know a transport for the
	// given multiaddr.
	ErrNoTransport = errors.New("no transport for protocol")

	// ErrAllDialsFailed is returned when connecting to a peer has ultimately failed
	ErrAllDialsFailed = errors.New("all dials failed")

	// ErrNoAddresses is returned when we fail to find any addresses for a
	// peer we're trying to dial.
	ErrNoAddresses = errors.New("no addresses")

	// ErrNoGoodAddresses is returned when we find addresses for a peer but
	// can't use any of them.
	ErrNoGoodAddresses = errors.New("no good addresses")

	// ErrGaterDisallowedConnection is returned when the gater prevents us from
	// forming a connection with a peer.
	ErrGaterDisallowedConnection = errors.New("gater disallows connection to peer")
)

// DialAttempts governs how many times a goroutine will try to dial a given peer.
// Note: this is down to one, as we have _too many dials_ atm. To add back in,
// add loop back in Dial(.)
const DialAttempts = 1

// ConcurrentFdDials is the number of concurrent outbound dials over transports
// that consume file descriptors
const ConcurrentFdDials = 160

// DefaultPerPeerRateLimit is the number of concurrent outbound dials to make
// per peer
const DefaultPerPeerRateLimit = 8

// dialbackoff is a struct used to avoid over-dialing the same, dead peers.
// Whenever we totally time out on a peer (all three attempts), we add them
// to dialbackoff. Then, whenevers goroutines would _wait_ (dialsync), they
// check dialbackoff. If it's there, they don't wait and exit promptly with
// an error. (the single goroutine that is actually dialing continues to
// dial). If a dial is successful, the peer is removed from backoff.
// Example:
//
//  for {
//  	if ok, wait := dialsync.Lock(p); !ok {
//  		if backoff.Backoff(p) {
//  			return errDialFailed
//  		}
//  		<-wait
//  		continue
//  	}
//  	defer dialsync.Unlock(p)
//  	c, err := actuallyDial(p)
//  	if err != nil {
//  		dialbackoff.AddBackoff(p)
//  		continue
//  	}
//  	dialbackoff.Clear(p)
//  }
//

// DialBackoff is a type for tracking peer dial backoffs.
//
// * It's safe to use its zero value.
// * It's thread-safe.
// * It's *not* safe to move this type after using.
type DialBackoff struct {
	entries map[peer.ID]map[string]*backoffAddr
	lock    sync.RWMutex
}

type backoffAddr struct {
	tries int
	until time.Time
}

func (db *DialBackoff) init(ctx context.Context) {
	if db.entries == nil {
		db.entries = make(map[peer.ID]map[string]*backoffAddr)
	}
	go db.background(ctx)
}

func (db *DialBackoff) background(ctx context.Context) {
	ticker := time.NewTicker(BackoffMax)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			db.cleanup()
		}
	}
}

// Backoff returns whether the client should backoff from dialing
// peer p at address addr
func (db *DialBackoff) Backoff(p peer.ID, addr ma.Multiaddr) (backoff bool) {
	db.lock.Lock()
	defer db.lock.Unlock()

	ap, found := db.entries[p][string(addr.Bytes())]
	return found && time.Now().Before(ap.until)
}

// BackoffBase is the base amount of time to backoff (default: 5s).
var BackoffBase = time.Second * 5

// BackoffCoef is the backoff coefficient (default: 1s).
var BackoffCoef = time.Second

// BackoffMax is the maximum backoff time (default: 5m).
var BackoffMax = time.Minute * 5

// AddBackoff lets other nodes know that we've entered backoff with
// peer p, so dialers should not wait unnecessarily. We still will
// attempt to dial with one goroutine, in case we get through.
//
// Backoff is not exponential, it's quadratic and computed according to the
// following formula:
//
//     BackoffBase + BakoffCoef * PriorBackoffs^2
//
// Where PriorBackoffs is the number of previous backoffs.
func (db *DialBackoff) AddBackoff(p peer.ID, addr ma.Multiaddr) {
	saddr := string(addr.Bytes())
	db.lock.Lock()
	defer db.lock.Unlock()
	bp, ok := db.entries[p]
	if !ok {
		bp = make(map[string]*backoffAddr, 1)
		db.entries[p] = bp
	}
	ba, ok := bp[saddr]
	if !ok {
		bp[saddr] = &backoffAddr{
			tries: 1,
			until: time.Now().Add(BackoffBase),
		}
		return
	}

	backoffTime := BackoffBase + BackoffCoef*time.Duration(ba.tries*ba.tries)
	if backoffTime > BackoffMax {
		backoffTime = BackoffMax
	}
	ba.until = time.Now().Add(backoffTime)
	ba.tries++
}

// Clear removes a backoff record. Clients should call this after a
// successful Dial.
func (db *DialBackoff) Clear(p peer.ID) {
	db.lock.Lock()
	defer db.lock.Unlock()
	delete(db.entries, p)
}

func (db *DialBackoff) cleanup() {
	db.lock.Lock()
	defer db.lock.Unlock()
	now := time.Now()
	for p, e := range db.entries {
		good := false
		for _, backoff := range e {
			backoffTime := BackoffBase + BackoffCoef*time.Duration(backoff.tries*backoff.tries)
			if backoffTime > BackoffMax {
				backoffTime = BackoffMax
			}
			if now.Before(backoff.until.Add(backoffTime)) {
				good = true
				break
			}
		}
		if !good {
			delete(db.entries, p)
		}
	}
}

// DialPeer connects to a peer.
//
// The idea is that the client of Swarm does not need to know what network
// the connection will happen over. Swarm can use whichever it choses.
// This allows us to use various transport protocols, do NAT traversal/relay,
// etc. to achieve connection.
func (s *Swarm) DialPeer(ctx context.Context, p peer.ID) (network.Conn, error) {
	if s.gater != nil && !s.gater.InterceptPeerDial(p) {
		log.Debugf("gater disallowed outbound connection to peer %s", p.Pretty())
		return nil, &DialError{Peer: p, Cause: ErrGaterDisallowedConnection}
	}

	// Avoid typed nil issues.
	c, err := s.dialPeer(ctx, p)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// internal dial method that returns an unwrapped conn
//
// It is gated by the swarm's dial synchronization systems: dialsync and
// dialbackoff.
func (s *Swarm) dialPeer(ctx context.Context, p peer.ID) (*Conn, error) {
	log.Debugw("dialing peer", "from", s.local, "to", p)
	err := p.Validate()
	if err != nil {
		return nil, err
	}

	if p == s.local {
		return nil, ErrDialToSelf
	}

	// check if we already have an open (usable) connection first
	conn := s.bestAcceptableConnToPeer(ctx, p)
	if conn != nil {
		return conn, nil
	}

	// apply the DialPeer timeout
	ctx, cancel := context.WithTimeout(ctx, network.GetDialPeerTimeout(ctx))
	defer cancel()

	conn, err = s.dsync.Dial(ctx, p)
	if err == nil {
		return conn, nil
	}

	log.Debugf("network for %s finished dialing %s", s.local, p)

	if ctx.Err() != nil {
		// Context error trumps any dial errors as it was likely the ultimate cause.
		return nil, ctx.Err()
	}

	if s.ctx.Err() != nil {
		// Ok, so the swarm is shutting down.
		return nil, ErrSwarmClosed
	}

	return nil, err
}

///////////////////////////////////////////////////////////////////////////////////
// lo and behold, The Dialer
// TODO explain how all this works
//////////////////////////////////////////////////////////////////////////////////

type dialRequest struct {
	ctx   context.Context
	resch chan dialResponse
}

type dialResponse struct {
	conn *Conn
	err  error
}

// dialWorkerLoop synchronizes and executes concurrent dials to a single peer
func (s *Swarm) dialWorkerLoop(p peer.ID, reqch <-chan dialRequest) {
	defer s.limiter.clearAllPeerDials(p)

	type pendRequest struct {
		req   dialRequest               // the original request
		err   *DialError                // dial error accumulator
		addrs map[ma.Multiaddr]struct{} // pending addr dials
	}

	type addrDial struct {
		addr     ma.Multiaddr
		ctx      context.Context
		conn     *Conn
		err      error
		requests []int
		dialed   bool
	}

	reqno := 0
	requests := make(map[int]*pendRequest)
	pending := make(map[ma.Multiaddr]*addrDial)

	dispatchError := func(ad *addrDial, err error) {
		ad.err = err
		for _, reqno := range ad.requests {
			pr, ok := requests[reqno]
			if !ok {
				// has already been dispatched
				continue
			}

			// accumulate the error
			pr.err.recordErr(ad.addr, err)

			delete(pr.addrs, ad.addr)
			if len(pr.addrs) == 0 {
				// all addrs have erred, dispatch dial error
				// but first do a last one check in case an acceptable connection has landed from
				// a simultaneous dial that started later and added new acceptable addrs
				c := s.bestAcceptableConnToPeer(pr.req.ctx, p)
				if c != nil {
					pr.req.resch <- dialResponse{conn: c}
				} else {
					pr.req.resch <- dialResponse{err: pr.err}
				}
				delete(requests, reqno)
			}
		}

		ad.requests = nil

		// if it was a backoff, clear the address dial so that it doesn't inhibit new dial requests.
		// this is necessary to support active listen scenarios, where a new dial comes in while
		// another dial is in progress, and needs to do a direct connection without inhibitions from
		// dial backoff.
		// it is also necessary to preserve consisent behaviour with the old dialer -- TestDialBackoff
		// regresses without this.
		if err == ErrDialBackoff {
			delete(pending, ad.addr)
		}
	}

	var triggerDial <-chan struct{}
	triggerNow := make(chan struct{})
	close(triggerNow)

	var nextDial []ma.Multiaddr
	active := 0
	done := false      // true when the request channel has been closed
	connected := false // true when a connection has been successfully established

	resch := make(chan dialResult)

loop:
	for {
		select {
		case req, ok := <-reqch:
			if !ok {
				// request channel has been closed, wait for pending dials to complete
				if active > 0 {
					done = true
					reqch = nil
					triggerDial = nil
					continue loop
				}

				// no active dials, we are done
				return
			}

			c := s.bestAcceptableConnToPeer(req.ctx, p)
			if c != nil {
				req.resch <- dialResponse{conn: c}
				continue loop
			}

			addrs, err := s.addrsForDial(req.ctx, p)
			if err != nil {
				req.resch <- dialResponse{err: err}
				continue loop
			}

			// at this point, len(addrs) > 0 or else it would be error from addrsForDial
			// ranke them to process in order
			addrs = s.rankAddrs(addrs)

			// create the pending request object
			pr := &pendRequest{
				req:   req,
				err:   &DialError{Peer: p},
				addrs: make(map[ma.Multiaddr]struct{}),
			}
			for _, a := range addrs {
				pr.addrs[a] = struct{}{}
			}

			// check if any of the addrs has been successfully dialed and accumulate
			// errors from complete dials while collecting new addrs to dial/join
			var todial []ma.Multiaddr
			var tojoin []*addrDial

			for _, a := range addrs {
				ad, ok := pending[a]
				if !ok {
					todial = append(todial, a)
					continue
				}

				if ad.conn != nil {
					// dial to this addr was successful, complete the request
					req.resch <- dialResponse{conn: ad.conn}
					continue loop
				}

				if ad.err != nil {
					// dial to this addr errored, accumulate the error
					pr.err.recordErr(a, ad.err)
					delete(pr.addrs, a)
					continue
				}

				// dial is still pending, add to the join list
				tojoin = append(tojoin, ad)
			}

			if len(todial) == 0 && len(tojoin) == 0 {
				// all request applicable addrs have been dialed, we must have errored
				req.resch <- dialResponse{err: pr.err}
				continue loop
			}

			// the request has some pending or new dials, track it and schedule new dials
			reqno++
			requests[reqno] = pr

			for _, ad := range tojoin {
				if !ad.dialed {
					if simConnect, isClient, reason := network.GetSimultaneousConnect(req.ctx); simConnect {
						if simConnect, _, _ := network.GetSimultaneousConnect(ad.ctx); !simConnect {
							ad.ctx = network.WithSimultaneousConnect(ad.ctx, isClient, reason)
						}
					}
				}
				ad.requests = append(ad.requests, reqno)
			}

			if len(todial) > 0 {
				for _, a := range todial {
					pending[a] = &addrDial{addr: a, ctx: req.ctx, requests: []int{reqno}}
				}

				nextDial = append(nextDial, todial...)
				nextDial = s.rankAddrs(nextDial)

				// trigger a new dial now to account for the new addrs we added
				triggerDial = triggerNow
			}

		case <-triggerDial:
			for _, addr := range nextDial {
				// spawn the dial
				ad := pending[addr]
				err := s.dialNextAddr(ad.ctx, p, addr, resch)
				if err != nil {
					dispatchError(ad, err)
				}
			}

			nextDial = nil
			triggerDial = nil

		case res := <-resch:
			active--

			if res.Conn != nil {
				connected = true
			}

			if done && active == 0 {
				if res.Conn != nil {
					// we got an actual connection, but the dial has been cancelled
					// Should we close it? I think not, we should just add it to the swarm
					_, err := s.addConn(res.Conn, network.DirOutbound)
					if err != nil {
						// well duh, now we have to close it
						res.Conn.Close()
					}
				}
				return
			}

			ad := pending[res.Addr]

			if res.Conn != nil {
				// we got a connection, add it to the swarm
				conn, err := s.addConn(res.Conn, network.DirOutbound)
				if err != nil {
					// oops no, we failed to add it to the swarm
					res.Conn.Close()
					dispatchError(ad, err)
					if active == 0 && len(nextDial) > 0 {
						triggerDial = triggerNow
					}
					continue loop
				}

				// dispatch to still pending requests
				for _, reqno := range ad.requests {
					pr, ok := requests[reqno]
					if !ok {
						// it has already dispatched a connection
						continue
					}

					pr.req.resch <- dialResponse{conn: conn}
					delete(requests, reqno)
				}

				ad.conn = conn
				ad.requests = nil

				continue loop
			}

			// it must be an error -- add backoff if applicable and dispatch
			if res.Err != context.Canceled && !connected {
				// we only add backoff if there has not been a successful connection
				// for consistency with the old dialer behavior.
				s.backf.AddBackoff(p, res.Addr)
			}

			dispatchError(ad, res.Err)
			if active == 0 && len(nextDial) > 0 {
				triggerDial = triggerNow
			}
		}
	}
}

func (s *Swarm) addrsForDial(ctx context.Context, p peer.ID) ([]ma.Multiaddr, error) {
	peerAddrs := s.peers.Addrs(p)
	if len(peerAddrs) == 0 {
		return nil, ErrNoAddresses
	}

	goodAddrs := s.filterKnownUndialables(p, peerAddrs)
	if forceDirect, _ := network.GetForceDirectDial(ctx); forceDirect {
		goodAddrs = addrutil.FilterAddrs(goodAddrs, s.nonProxyAddr)
	}

	if len(goodAddrs) == 0 {
		return nil, ErrNoGoodAddresses
	}

	return goodAddrs, nil
}

func (s *Swarm) dialNextAddr(ctx context.Context, p peer.ID, addr ma.Multiaddr, resch chan dialResult) error {
	// check the dial backoff
	if forceDirect, _ := network.GetForceDirectDial(ctx); !forceDirect {
		if s.backf.Backoff(p, addr) {
			return ErrDialBackoff
		}
	}

	// start the dial
	s.limitedDial(ctx, p, addr, resch)

	return nil
}

func (s *Swarm) canDial(addr ma.Multiaddr) bool {
	t := s.TransportForDialing(addr)
	return t != nil && t.CanDial(addr)
}

func (s *Swarm) nonProxyAddr(addr ma.Multiaddr) bool {
	t := s.TransportForDialing(addr)
	return !t.Proxy()
}

// ranks addresses in descending order of preference for dialing, with the following rules:
// NonRelay > Relay
// NonWS > WS
// Private > Public
// UDP > TCP
func (s *Swarm) rankAddrs(addrs []ma.Multiaddr) []ma.Multiaddr {
	addrTier := func(a ma.Multiaddr) (tier int) {
		if isRelayAddr(a) {
			tier |= 0b1000
		}
		if isExpensiveAddr(a) {
			tier |= 0b0100
		}
		if !manet.IsPrivateAddr(a) {
			tier |= 0b0010
		}
		if isFdConsumingAddr(a) {
			tier |= 0b0001
		}

		return tier
	}

	tiers := make([][]ma.Multiaddr, 16)
	for _, a := range addrs {
		tier := addrTier(a)
		tiers[tier] = append(tiers[tier], a)
	}

	result := make([]ma.Multiaddr, 0, len(addrs))
	for _, tier := range tiers {
		result = append(result, tier...)
	}

	return result
}

// filterKnownUndialables takes a list of multiaddrs, and removes those
// that we definitely don't want to dial: addresses configured to be blocked,
// IPv6 link-local addresses, addresses without a dial-capable transport,
// and addresses that we know to be our own.
// This is an optimization to avoid wasting time on dials that we know are going to fail.
func (s *Swarm) filterKnownUndialables(p peer.ID, addrs []ma.Multiaddr) []ma.Multiaddr {
	lisAddrs, _ := s.InterfaceListenAddresses()
	var ourAddrs []ma.Multiaddr
	for _, addr := range lisAddrs {
		protos := addr.Protocols()
		// we're only sure about filtering out /ip4 and /ip6 addresses, so far
		if protos[0].Code == ma.P_IP4 || protos[0].Code == ma.P_IP6 {
			ourAddrs = append(ourAddrs, addr)
		}
	}

	return addrutil.FilterAddrs(addrs,
		addrutil.SubtractFilter(ourAddrs...),
		s.canDial,
		// TODO: Consider allowing link-local addresses
		addrutil.AddrOverNonLocalIP,
		func(addr ma.Multiaddr) bool {
			return s.gater == nil || s.gater.InterceptAddrDial(p, addr)
		},
	)
}

// limitedDial will start a dial to the given peer when
// it is able, respecting the various different types of rate
// limiting that occur without using extra goroutines per addr
func (s *Swarm) limitedDial(ctx context.Context, p peer.ID, a ma.Multiaddr, resp chan dialResult) {
	s.limiter.AddDialJob(&dialJob{
		addr: a,
		peer: p,
		resp: resp,
		ctx:  ctx,
	})
}

// dialAddr is the actual dial for an addr, indirectly invoked through the limiter
func (s *Swarm) dialAddr(ctx context.Context, p peer.ID, addr ma.Multiaddr) (transport.CapableConn, error) {
	// Just to double check. Costs nothing.
	if s.local == p {
		return nil, ErrDialToSelf
	}
	log.Debugf("%s swarm dialing %s %s", s.local, p, addr)

	tpt := s.TransportForDialing(addr)
	if tpt == nil {
		return nil, ErrNoTransport
	}

	connC, err := tpt.Dial(ctx, addr, p)
	if err != nil {
		return nil, err
	}

	// Trust the transport? Yeah... right.
	if connC.RemotePeer() != p {
		connC.Close()
		err = fmt.Errorf("BUG in transport %T: tried to dial %s, dialed %s", p, connC.RemotePeer(), tpt)
		log.Error(err)
		return nil, err
	}

	// success! we got one!
	return connC, nil
}

// TODO We should have a `IsFdConsuming() bool` method on the `Transport` interface in go-libp2p-core/transport.
// This function checks if any of the transport protocols in the address requires a file descriptor.
// For now:
// A Non-circuit address which has the TCP/UNIX protocol is deemed FD consuming.
// For a circuit-relay address, we look at the address of the relay server/proxy
// and use the same logic as above to decide.
func isFdConsumingAddr(addr ma.Multiaddr) bool {
	first, _ := ma.SplitFunc(addr, func(c ma.Component) bool {
		return c.Protocol().Code == ma.P_CIRCUIT
	})

	// for safety
	if first == nil {
		return true
	}

	_, err1 := first.ValueForProtocol(ma.P_TCP)
	_, err2 := first.ValueForProtocol(ma.P_UNIX)
	return err1 == nil || err2 == nil
}

func isExpensiveAddr(addr ma.Multiaddr) bool {
	_, err1 := addr.ValueForProtocol(ma.P_WS)
	_, err2 := addr.ValueForProtocol(ma.P_WSS)
	return err1 == nil || err2 == nil
}

func isRelayAddr(addr ma.Multiaddr) bool {
	_, err := addr.ValueForProtocol(ma.P_CIRCUIT)
	return err == nil
}
