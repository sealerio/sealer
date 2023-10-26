package messagequeue

import (
	"context"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

const (
	// dontHaveTimeout is used to simulate a DONT_HAVE when communicating with
	// a peer whose Bitswap client doesn't support the DONT_HAVE response,
	// or when the peer takes too long to respond.
	// If the peer doesn't respond to a want-block within the timeout, the
	// local node assumes that the peer doesn't have the block.
	dontHaveTimeout = 5 * time.Second

	// maxExpectedWantProcessTime is the maximum amount of time we expect a
	// peer takes to process a want and initiate sending a response to us
	maxExpectedWantProcessTime = 2 * time.Second

	// maxTimeout is the maximum allowed timeout, regardless of latency
	maxTimeout = dontHaveTimeout + maxExpectedWantProcessTime

	// pingLatencyMultiplier is multiplied by the average ping time to
	// get an upper bound on how long we expect to wait for a peer's response
	// to arrive
	pingLatencyMultiplier = 3

	// messageLatencyAlpha is the alpha supplied to the message latency EWMA
	messageLatencyAlpha = 0.5

	// To give a margin for error, the timeout is calculated as
	// messageLatencyMultiplier * message latency
	messageLatencyMultiplier = 2
)

// PeerConnection is a connection to a peer that can be pinged, and the
// average latency measured
type PeerConnection interface {
	// Ping the peer
	Ping(context.Context) ping.Result
	// The average latency of all pings
	Latency() time.Duration
}

// pendingWant keeps track of a want that has been sent and we're waiting
// for a response or for a timeout to expire
type pendingWant struct {
	c      cid.Cid
	active bool
	sent   time.Time
}

// dontHaveTimeoutMgr simulates a DONT_HAVE message if the peer takes too long
// to respond to a message.
// The timeout is based on latency - we start with a default latency, while
// we ping the peer to estimate latency. If we receive a response from the
// peer we use the response latency.
type dontHaveTimeoutMgr struct {
	clock                      clock.Clock
	ctx                        context.Context
	shutdown                   func()
	peerConn                   PeerConnection
	onDontHaveTimeout          func([]cid.Cid)
	defaultTimeout             time.Duration
	maxTimeout                 time.Duration
	pingLatencyMultiplier      int
	messageLatencyMultiplier   int
	maxExpectedWantProcessTime time.Duration

	// All variables below here must be protected by the lock
	lk sync.RWMutex
	// has the timeout manager started
	started bool
	// wants that are active (waiting for a response or timeout)
	activeWants map[cid.Cid]*pendingWant
	// queue of wants, from oldest to newest
	wantQueue []*pendingWant
	// time to wait for a response (depends on latency)
	timeout time.Duration
	// ewma of message latency (time from message sent to response received)
	messageLatency *latencyEwma
	// timer used to wait until want at front of queue expires
	checkForTimeoutsTimer *clock.Timer
	// used for testing -- timeoutsTriggered when a scheduled dont have timeouts were triggered
	timeoutsTriggered chan struct{}
}

// newDontHaveTimeoutMgr creates a new dontHaveTimeoutMgr
// onDontHaveTimeout is called when pending keys expire (not cancelled before timeout)
func newDontHaveTimeoutMgr(pc PeerConnection, onDontHaveTimeout func([]cid.Cid), clock clock.Clock) *dontHaveTimeoutMgr {
	return newDontHaveTimeoutMgrWithParams(pc, onDontHaveTimeout, dontHaveTimeout, maxTimeout,
		pingLatencyMultiplier, messageLatencyMultiplier, maxExpectedWantProcessTime, clock, nil)
}

// newDontHaveTimeoutMgrWithParams is used by the tests
func newDontHaveTimeoutMgrWithParams(
	pc PeerConnection,
	onDontHaveTimeout func([]cid.Cid),
	defaultTimeout time.Duration,
	maxTimeout time.Duration,
	pingLatencyMultiplier int,
	messageLatencyMultiplier int,
	maxExpectedWantProcessTime time.Duration,
	clock clock.Clock,
	timeoutsTriggered chan struct{}) *dontHaveTimeoutMgr {

	ctx, shutdown := context.WithCancel(context.Background())
	mqp := &dontHaveTimeoutMgr{
		clock:                      clock,
		ctx:                        ctx,
		shutdown:                   shutdown,
		peerConn:                   pc,
		activeWants:                make(map[cid.Cid]*pendingWant),
		timeout:                    defaultTimeout,
		messageLatency:             &latencyEwma{alpha: messageLatencyAlpha},
		defaultTimeout:             defaultTimeout,
		maxTimeout:                 maxTimeout,
		pingLatencyMultiplier:      pingLatencyMultiplier,
		messageLatencyMultiplier:   messageLatencyMultiplier,
		maxExpectedWantProcessTime: maxExpectedWantProcessTime,
		onDontHaveTimeout:          onDontHaveTimeout,
		timeoutsTriggered:          timeoutsTriggered,
	}

	return mqp
}

// Shutdown the dontHaveTimeoutMgr. Any subsequent call to Start() will be ignored
func (dhtm *dontHaveTimeoutMgr) Shutdown() {
	dhtm.shutdown()

	dhtm.lk.Lock()
	defer dhtm.lk.Unlock()

	// Clear any pending check for timeouts
	if dhtm.checkForTimeoutsTimer != nil {
		dhtm.checkForTimeoutsTimer.Stop()
	}
}

// Start the dontHaveTimeoutMgr. This method is idempotent
func (dhtm *dontHaveTimeoutMgr) Start() {
	dhtm.lk.Lock()
	defer dhtm.lk.Unlock()

	// Make sure the dont have timeout manager hasn't already been started
	if dhtm.started {
		return
	}
	dhtm.started = true

	// If we already have a measure of latency to the peer, use it to
	// calculate a reasonable timeout
	latency := dhtm.peerConn.Latency()
	if latency.Nanoseconds() > 0 {
		dhtm.timeout = dhtm.calculateTimeoutFromPingLatency(latency)
		return
	}

	// Otherwise measure latency by pinging the peer
	go dhtm.measurePingLatency()
}

// UpdateMessageLatency is called when we receive a response from the peer.
// It is the time between sending a request and receiving the corresponding
// response.
func (dhtm *dontHaveTimeoutMgr) UpdateMessageLatency(elapsed time.Duration) {
	dhtm.lk.Lock()
	defer dhtm.lk.Unlock()

	// Update the message latency and the timeout
	dhtm.messageLatency.update(elapsed)
	oldTimeout := dhtm.timeout
	dhtm.timeout = dhtm.calculateTimeoutFromMessageLatency()

	// If the timeout has decreased
	if dhtm.timeout < oldTimeout {
		// Check if after changing the timeout there are any pending wants that
		// are now over the timeout
		dhtm.checkForTimeouts()
	}
}

// measurePingLatency measures the latency to the peer by pinging it
func (dhtm *dontHaveTimeoutMgr) measurePingLatency() {
	// Wait up to defaultTimeout for a response to the ping
	ctx, cancel := context.WithTimeout(dhtm.ctx, dhtm.defaultTimeout)
	defer cancel()

	// Ping the peer
	res := dhtm.peerConn.Ping(ctx)
	if res.Error != nil {
		// If there was an error, we'll just leave the timeout as
		// defaultTimeout
		return
	}

	// Get the average latency to the peer
	latency := dhtm.peerConn.Latency()

	dhtm.lk.Lock()
	defer dhtm.lk.Unlock()

	// A message has arrived so we already set the timeout based on message latency
	if dhtm.messageLatency.samples > 0 {
		return
	}

	// Calculate a reasonable timeout based on latency
	dhtm.timeout = dhtm.calculateTimeoutFromPingLatency(latency)

	// Check if after changing the timeout there are any pending wants that are
	// now over the timeout
	dhtm.checkForTimeouts()
}

// checkForTimeouts checks pending wants to see if any are over the timeout.
// Note: this function should only be called within the lock.
func (dhtm *dontHaveTimeoutMgr) checkForTimeouts() {

	if len(dhtm.wantQueue) == 0 {
		return
	}

	// Figure out which of the blocks that were wanted were not received
	// within the timeout
	expired := make([]cid.Cid, 0, len(dhtm.activeWants))
	for len(dhtm.wantQueue) > 0 {
		pw := dhtm.wantQueue[0]

		// If the want is still active
		if pw.active {
			// The queue is in order from earliest to latest, so if we
			// didn't find an expired entry we can stop iterating
			if dhtm.clock.Since(pw.sent) < dhtm.timeout {
				break
			}

			// Add the want to the expired list
			expired = append(expired, pw.c)
			// Remove the want from the activeWants map
			delete(dhtm.activeWants, pw.c)
		}

		// Remove expired or cancelled wants from the want queue
		dhtm.wantQueue = dhtm.wantQueue[1:]
	}

	// Fire the timeout event for the expired wants
	if len(expired) > 0 {
		go dhtm.fireTimeout(expired)
	}

	if len(dhtm.wantQueue) == 0 {
		return
	}

	// Make sure the timeout manager is still running
	if dhtm.ctx.Err() != nil {
		return
	}

	// Schedule the next check for the moment when the oldest pending want will
	// timeout
	oldestStart := dhtm.wantQueue[0].sent
	until := oldestStart.Add(dhtm.timeout).Sub(dhtm.clock.Now())
	if dhtm.checkForTimeoutsTimer == nil {
		dhtm.checkForTimeoutsTimer = dhtm.clock.Timer(until)
		go dhtm.consumeTimeouts()
	} else {
		dhtm.checkForTimeoutsTimer.Stop()
		dhtm.checkForTimeoutsTimer.Reset(until)
	}
}

func (dhtm *dontHaveTimeoutMgr) consumeTimeouts() {
	for {
		select {
		case <-dhtm.ctx.Done():
			return
		case <-dhtm.checkForTimeoutsTimer.C:
			dhtm.lk.Lock()
			dhtm.checkForTimeouts()
			dhtm.lk.Unlock()
		}
	}
}

// AddPending adds the given keys that will expire if not cancelled before
// the timeout
func (dhtm *dontHaveTimeoutMgr) AddPending(ks []cid.Cid) {
	if len(ks) == 0 {
		return
	}

	start := dhtm.clock.Now()

	dhtm.lk.Lock()
	defer dhtm.lk.Unlock()

	queueWasEmpty := len(dhtm.activeWants) == 0

	// Record the start time for each key
	for _, c := range ks {
		if _, ok := dhtm.activeWants[c]; !ok {
			pw := pendingWant{
				c:      c,
				sent:   start,
				active: true,
			}
			dhtm.activeWants[c] = &pw
			dhtm.wantQueue = append(dhtm.wantQueue, &pw)
		}
	}

	// If there was already an earlier pending item in the queue, then there
	// must already be a timeout check scheduled. If there is nothing in the
	// queue then we should make sure to schedule a check.
	if queueWasEmpty {
		dhtm.checkForTimeouts()
	}
}

// CancelPending is called when we receive a response for a key
func (dhtm *dontHaveTimeoutMgr) CancelPending(ks []cid.Cid) {
	dhtm.lk.Lock()
	defer dhtm.lk.Unlock()

	// Mark the wants as cancelled
	for _, c := range ks {
		if pw, ok := dhtm.activeWants[c]; ok {
			pw.active = false
			delete(dhtm.activeWants, c)
		}
	}
}

// fireTimeout fires the onDontHaveTimeout method with the timed out keys
func (dhtm *dontHaveTimeoutMgr) fireTimeout(pending []cid.Cid) {
	// Make sure the timeout manager has not been shut down
	if dhtm.ctx.Err() != nil {
		return
	}

	// Fire the timeout
	dhtm.onDontHaveTimeout(pending)

	// signal a timeout fired
	if dhtm.timeoutsTriggered != nil {
		dhtm.timeoutsTriggered <- struct{}{}
	}
}

// calculateTimeoutFromPingLatency calculates a reasonable timeout derived from latency
func (dhtm *dontHaveTimeoutMgr) calculateTimeoutFromPingLatency(latency time.Duration) time.Duration {
	// The maximum expected time for a response is
	// the expected time to process the want + (latency * multiplier)
	// The multiplier is to provide some padding for variable latency.
	timeout := dhtm.maxExpectedWantProcessTime + time.Duration(dhtm.pingLatencyMultiplier)*latency
	if timeout > dhtm.maxTimeout {
		timeout = dhtm.maxTimeout
	}
	return timeout
}

// calculateTimeoutFromMessageLatency calculates a timeout derived from message latency
func (dhtm *dontHaveTimeoutMgr) calculateTimeoutFromMessageLatency() time.Duration {
	timeout := dhtm.messageLatency.latency * time.Duration(dhtm.messageLatencyMultiplier)
	if timeout > dhtm.maxTimeout {
		timeout = dhtm.maxTimeout
	}
	return timeout
}

// latencyEwma is an EWMA of message latency
type latencyEwma struct {
	alpha   float64
	samples uint64
	latency time.Duration
}

// update the EWMA with the given sample
func (le *latencyEwma) update(elapsed time.Duration) {
	le.samples++

	// Initially set alpha to be 1.0 / <the number of samples>
	alpha := 1.0 / float64(le.samples)
	if alpha < le.alpha {
		// Once we have enough samples, clamp alpha
		alpha = le.alpha
	}
	le.latency = time.Duration(float64(elapsed)*alpha + (1-alpha)*float64(le.latency))
}
