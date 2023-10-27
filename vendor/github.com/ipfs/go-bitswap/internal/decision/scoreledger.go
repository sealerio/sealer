package decision

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

const (
	// the alpha for the EWMA used to track short term usefulness
	shortTermAlpha = 0.5

	// the alpha for the EWMA used to track long term usefulness
	longTermAlpha = 0.05

	// how frequently the engine should sample usefulness. Peers that
	// interact every shortTerm time period are considered "active".
	shortTerm = 10 * time.Second

	// long term ratio defines what "long term" means in terms of the
	// shortTerm duration. Peers that interact once every longTermRatio are
	// considered useful over the long term.
	longTermRatio = 10

	// long/short term scores for tagging peers
	longTermScore  = 10 // this is a high tag but it grows _very_ slowly.
	shortTermScore = 10 // this is a high tag but it'll go away quickly if we aren't using the peer.
)

// Stores the data exchange relationship between two peers.
type scoreledger struct {
	// Partner is the remote Peer.
	partner peer.ID

	// tracks bytes sent...
	bytesSent uint64

	// ...and received.
	bytesRecv uint64

	// lastExchange is the time of the last data exchange.
	lastExchange time.Time

	// These scores keep track of how useful we think this peer is. Short
	// tracks short-term usefulness and long tracks long-term usefulness.
	shortScore, longScore float64

	// Score keeps track of the score used in the peer tagger. We track it
	// here to avoid unnecessarily updating the tags in the connection manager.
	score int

	// exchangeCount is the number of exchanges with this peer
	exchangeCount uint64

	// the record lock
	lock sync.RWMutex

	clock clock.Clock
}

// Receipt is a summary of the ledger for a given peer
// collecting various pieces of aggregated data for external
// reporting purposes.
type Receipt struct {
	Peer      string
	Value     float64
	Sent      uint64
	Recv      uint64
	Exchanged uint64
}

// Increments the sent counter.
func (l *scoreledger) AddToSentBytes(n int) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.exchangeCount++
	l.lastExchange = l.clock.Now()
	l.bytesSent += uint64(n)
}

// Increments the received counter.
func (l *scoreledger) AddToReceivedBytes(n int) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.exchangeCount++
	l.lastExchange = l.clock.Now()
	l.bytesRecv += uint64(n)
}

// Returns the Receipt for this ledger record.
func (l *scoreledger) Receipt() *Receipt {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return &Receipt{
		Peer:      l.partner.String(),
		Value:     float64(l.bytesSent) / float64(l.bytesRecv+1),
		Sent:      l.bytesSent,
		Recv:      l.bytesRecv,
		Exchanged: l.exchangeCount,
	}
}

// DefaultScoreLedger is used by Engine as the default ScoreLedger.
type DefaultScoreLedger struct {
	// the score func
	scorePeer ScorePeerFunc
	// is closed on Close
	closing chan struct{}
	// protects the fields immediatly below
	lock sync.RWMutex
	// ledgerMap lists score ledgers by their partner key.
	ledgerMap map[peer.ID]*scoreledger
	// how frequently the engine should sample peer usefulness
	peerSampleInterval time.Duration
	// used by the tests to detect when a sample is taken
	sampleCh chan struct{}
	clock    clock.Clock
}

// scoreWorker keeps track of how "useful" our peers are, updating scores in the
// connection manager.
//
// It does this by tracking two scores: short-term usefulness and long-term
// usefulness. Short-term usefulness is sampled frequently and highly weights
// new observations. Long-term usefulness is sampled less frequently and highly
// weights on long-term trends.
//
// In practice, we do this by keeping two EWMAs. If we see an interaction
// within the sampling period, we record the score, otherwise, we record a 0.
// The short-term one has a high alpha and is sampled every shortTerm period.
// The long-term one has a low alpha and is sampled every
// longTermRatio*shortTerm period.
//
// To calculate the final score, we sum the short-term and long-term scores then
// adjust it ±25% based on our debt ratio. Peers that have historically been
// more useful to us than we are to them get the highest score.
func (dsl *DefaultScoreLedger) scoreWorker() {
	ticker := dsl.clock.Ticker(dsl.peerSampleInterval)
	defer ticker.Stop()

	type update struct {
		peer  peer.ID
		score int
	}
	var (
		lastShortUpdate, lastLongUpdate time.Time
		updates                         []update
	)

	for i := 0; ; i = (i + 1) % longTermRatio {
		var now time.Time
		select {
		case now = <-ticker.C:
		case <-dsl.closing:
			return
		}

		// The long term update ticks every `longTermRatio` short
		// intervals.
		updateLong := i == 0

		dsl.lock.Lock()
		for _, l := range dsl.ledgerMap {
			l.lock.Lock()

			// Update the short-term score.
			if l.lastExchange.After(lastShortUpdate) {
				l.shortScore = ewma(l.shortScore, shortTermScore, shortTermAlpha)
			} else {
				l.shortScore = ewma(l.shortScore, 0, shortTermAlpha)
			}

			// Update the long-term score.
			if updateLong {
				if l.lastExchange.After(lastLongUpdate) {
					l.longScore = ewma(l.longScore, longTermScore, longTermAlpha)
				} else {
					l.longScore = ewma(l.longScore, 0, longTermAlpha)
				}
			}

			// Calculate the new score.
			//
			// The accounting score adjustment prefers peers _we_
			// need over peers that need us. This doesn't help with
			// leeching.
			var lscore float64
			if l.bytesRecv == 0 {
				lscore = 0
			} else {
				lscore = float64(l.bytesRecv) / float64(l.bytesRecv+l.bytesSent)
			}
			score := int((l.shortScore + l.longScore) * (lscore*.5 + .75))

			// Avoid updating the connection manager unless there's a change. This can be expensive.
			if l.score != score {
				// put these in a list so we can perform the updates outside _global_ the lock.
				updates = append(updates, update{l.partner, score})
				l.score = score
			}
			l.lock.Unlock()
		}
		dsl.lock.Unlock()

		// record the times.
		lastShortUpdate = now
		if updateLong {
			lastLongUpdate = now
		}

		// apply the updates
		for _, update := range updates {
			dsl.scorePeer(update.peer, update.score)
		}
		// Keep the memory. It's not much and it saves us from having to allocate.
		updates = updates[:0]

		// Used by the tests
		if dsl.sampleCh != nil {
			dsl.sampleCh <- struct{}{}
		}
	}
}

// Returns the score ledger for the given peer or nil if that peer
// is not on the ledger.
func (dsl *DefaultScoreLedger) find(p peer.ID) *scoreledger {
	// Take a read lock (as it's less expensive) to check if we have
	// a ledger for the peer.
	dsl.lock.RLock()
	l, ok := dsl.ledgerMap[p]
	dsl.lock.RUnlock()
	if ok {
		return l
	}
	return nil
}

// Returns a new scoreledger.
func newScoreLedger(p peer.ID, clock clock.Clock) *scoreledger {
	return &scoreledger{
		partner: p,
		clock:   clock,
	}
}

// Lazily instantiates a ledger.
func (dsl *DefaultScoreLedger) findOrCreate(p peer.ID) *scoreledger {
	l := dsl.find(p)
	if l != nil {
		return l
	}

	// There's no ledger, so take a write lock, then check again and
	// create the ledger if necessary.
	dsl.lock.Lock()
	defer dsl.lock.Unlock()
	l, ok := dsl.ledgerMap[p]
	if !ok {
		l = newScoreLedger(p, dsl.clock)
		dsl.ledgerMap[p] = l
	}
	return l
}

// GetReceipt returns aggregated data communication with a given peer.
func (dsl *DefaultScoreLedger) GetReceipt(p peer.ID) *Receipt {
	l := dsl.find(p)
	if l != nil {
		return l.Receipt()
	}

	// Return a blank receipt otherwise.
	return &Receipt{
		Peer:      p.String(),
		Value:     0,
		Sent:      0,
		Recv:      0,
		Exchanged: 0,
	}
}

// Starts the default ledger sampling process.
func (dsl *DefaultScoreLedger) Start(scorePeer ScorePeerFunc) {
	dsl.init(scorePeer)
	go dsl.scoreWorker()
}

// Stops the sampling process.
func (dsl *DefaultScoreLedger) Stop() {
	close(dsl.closing)
}

// Initializes the score ledger.
func (dsl *DefaultScoreLedger) init(scorePeer ScorePeerFunc) {
	dsl.lock.Lock()
	defer dsl.lock.Unlock()
	dsl.scorePeer = scorePeer
}

// Increments the sent counter for the given peer.
func (dsl *DefaultScoreLedger) AddToSentBytes(p peer.ID, n int) {
	l := dsl.findOrCreate(p)
	l.AddToSentBytes(n)
}

// Increments the received counter for the given peer.
func (dsl *DefaultScoreLedger) AddToReceivedBytes(p peer.ID, n int) {
	l := dsl.findOrCreate(p)
	l.AddToReceivedBytes(n)
}

// PeerConnected should be called when a new peer connects, meaning
// we should open accounting.
func (dsl *DefaultScoreLedger) PeerConnected(p peer.ID) {
	dsl.lock.Lock()
	defer dsl.lock.Unlock()
	_, ok := dsl.ledgerMap[p]
	if !ok {
		dsl.ledgerMap[p] = newScoreLedger(p, dsl.clock)
	}
}

// PeerDisconnected should be called when a peer disconnects to
// clean up the accounting.
func (dsl *DefaultScoreLedger) PeerDisconnected(p peer.ID) {
	dsl.lock.Lock()
	defer dsl.lock.Unlock()
	delete(dsl.ledgerMap, p)
}

// Creates a new instance of the default score ledger.
func NewDefaultScoreLedger() *DefaultScoreLedger {
	return &DefaultScoreLedger{
		ledgerMap:          make(map[peer.ID]*scoreledger),
		closing:            make(chan struct{}),
		peerSampleInterval: shortTerm,
		clock:              clock.New(),
	}
}

// Creates a new instance of the default score ledger with testing
// parameters.
func NewTestScoreLedger(peerSampleInterval time.Duration, sampleCh chan struct{}, clock clock.Clock) *DefaultScoreLedger {
	dsl := NewDefaultScoreLedger()
	dsl.peerSampleInterval = peerSampleInterval
	dsl.sampleCh = sampleCh
	dsl.clock = clock
	return dsl
}
