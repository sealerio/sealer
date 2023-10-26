// Package bitswap implements the IPFS exchange interface with the BitSwap
// bilateral exchange protocol.
package bitswap

import (
	"context"
	"errors"
	"fmt"

	"sync"
	"time"

	delay "github.com/ipfs/go-ipfs-delay"

	deciface "github.com/ipfs/go-bitswap/decision"
	bsbpm "github.com/ipfs/go-bitswap/internal/blockpresencemanager"
	"github.com/ipfs/go-bitswap/internal/decision"
	"github.com/ipfs/go-bitswap/internal/defaults"
	bsgetter "github.com/ipfs/go-bitswap/internal/getter"
	bsmq "github.com/ipfs/go-bitswap/internal/messagequeue"
	"github.com/ipfs/go-bitswap/internal/notifications"
	bspm "github.com/ipfs/go-bitswap/internal/peermanager"
	bspqm "github.com/ipfs/go-bitswap/internal/providerquerymanager"
	bssession "github.com/ipfs/go-bitswap/internal/session"
	bssim "github.com/ipfs/go-bitswap/internal/sessioninterestmanager"
	bssm "github.com/ipfs/go-bitswap/internal/sessionmanager"
	bsspm "github.com/ipfs/go-bitswap/internal/sessionpeermanager"
	bsmsg "github.com/ipfs/go-bitswap/message"
	bsnet "github.com/ipfs/go-bitswap/network"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-metrics-interface"
	process "github.com/jbenet/goprocess"
	procctx "github.com/jbenet/goprocess/context"
	"github.com/libp2p/go-libp2p-core/peer"
)

var log = logging.Logger("bitswap")
var sflog = log.Desugar()

var _ exchange.SessionExchange = (*Bitswap)(nil)

var (
	// HasBlockBufferSize is the buffer size of the channel for new blocks
	// that need to be provided. They should get pulled over by the
	// provideCollector even before they are actually provided.
	// TODO: Does this need to be this large givent that?
	HasBlockBufferSize    = 256
	provideKeysBufferSize = 2048
	provideWorkerMax      = 6

	// the 1<<18+15 is to observe old file chunks that are 1<<18 + 14 in size
	metricsBuckets = []float64{1 << 6, 1 << 10, 1 << 14, 1 << 18, 1<<18 + 15, 1 << 22}

	timeMetricsBuckets = []float64{1, 10, 30, 60, 90, 120, 600}
)

// Option defines the functional option type that can be used to configure
// bitswap instances
type Option func(*Bitswap)

// ProvideEnabled is an option for enabling/disabling provide announcements
func ProvideEnabled(enabled bool) Option {
	return func(bs *Bitswap) {
		bs.provideEnabled = enabled
	}
}

// ProviderSearchDelay overwrites the global provider search delay
func ProviderSearchDelay(newProvSearchDelay time.Duration) Option {
	return func(bs *Bitswap) {
		bs.provSearchDelay = newProvSearchDelay
	}
}

// RebroadcastDelay overwrites the global provider rebroadcast delay
func RebroadcastDelay(newRebroadcastDelay delay.D) Option {
	return func(bs *Bitswap) {
		bs.rebroadcastDelay = newRebroadcastDelay
	}
}

// EngineBlockstoreWorkerCount sets the number of worker threads used for
// blockstore operations in the decision engine
func EngineBlockstoreWorkerCount(count int) Option {
	if count <= 0 {
		panic(fmt.Sprintf("Engine blockstore worker count is %d but must be > 0", count))
	}
	return func(bs *Bitswap) {
		bs.engineBstoreWorkerCount = count
	}
}

// EngineTaskWorkerCount sets the number of worker threads used inside the engine
func EngineTaskWorkerCount(count int) Option {
	if count <= 0 {
		panic(fmt.Sprintf("Engine task worker count is %d but must be > 0", count))
	}
	return func(bs *Bitswap) {
		bs.engineTaskWorkerCount = count
	}
}

func TaskWorkerCount(count int) Option {
	if count <= 0 {
		panic(fmt.Sprintf("task worker count is %d but must be > 0", count))
	}
	return func(bs *Bitswap) {
		bs.taskWorkerCount = count
	}
}

// MaxOutstandingBytesPerPeer describes approximately how much work we are will to have outstanding to a peer at any
// given time. Setting it to 0 will disable any limiting.
func MaxOutstandingBytesPerPeer(count int) Option {
	if count < 0 {
		panic(fmt.Sprintf("max outstanding bytes per peer is %d but must be >= 0", count))
	}
	return func(bs *Bitswap) {
		bs.engineMaxOutstandingBytesPerPeer = count
	}
}

// SetSendDontHaves indicates what to do when the engine receives a want-block
// for a block that is not in the blockstore. Either
// - Send a DONT_HAVE message
// - Simply don't respond
// This option is only used for testing.
func SetSendDontHaves(send bool) Option {
	return func(bs *Bitswap) {
		bs.engineSetSendDontHaves = send
	}
}

// Configures the engine to use the given score decision logic.
func WithScoreLedger(scoreLedger deciface.ScoreLedger) Option {
	return func(bs *Bitswap) {
		bs.engineScoreLedger = scoreLedger
	}
}

func SetSimulateDontHavesOnTimeout(send bool) Option {
	return func(bs *Bitswap) {
		bs.simulateDontHavesOnTimeout = send
	}
}

type TaskInfo = decision.TaskInfo
type TaskComparator = decision.TaskComparator

// WithTaskComparator configures custom task prioritization logic.
func WithTaskComparator(comparator TaskComparator) Option {
	return func(bs *Bitswap) {
		bs.taskComparator = comparator
	}
}

// New initializes a BitSwap instance that communicates over the provided
// BitSwapNetwork. This function registers the returned instance as the network
// delegate. Runs until context is cancelled or bitswap.Close is called.
func New(parent context.Context, network bsnet.BitSwapNetwork,
	bstore blockstore.Blockstore, options ...Option) exchange.Interface {

	// important to use provided parent context (since it may include important
	// loggable data). It's probably not a good idea to allow bitswap to be
	// coupled to the concerns of the ipfs daemon in this way.
	//
	// FIXME(btc) Now that bitswap manages itself using a process, it probably
	// shouldn't accept a context anymore. Clients should probably use Close()
	// exclusively. We should probably find another way to share logging data
	ctx, cancelFunc := context.WithCancel(parent)
	ctx = metrics.CtxSubScope(ctx, "bitswap")
	dupHist := metrics.NewCtx(ctx, "recv_dup_blocks_bytes", "Summary of duplicate"+
		" data blocks recived").Histogram(metricsBuckets)
	allHist := metrics.NewCtx(ctx, "recv_all_blocks_bytes", "Summary of all"+
		" data blocks recived").Histogram(metricsBuckets)

	sentHistogram := metrics.NewCtx(ctx, "sent_all_blocks_bytes", "Histogram of blocks sent by"+
		" this bitswap").Histogram(metricsBuckets)

	sendTimeHistogram := metrics.NewCtx(ctx, "send_times", "Histogram of how long it takes to send messages"+
		" in this bitswap").Histogram(timeMetricsBuckets)

	pendingEngineGauge := metrics.NewCtx(ctx, "pending_tasks", "Total number of pending tasks").Gauge()

	activeEngineGauge := metrics.NewCtx(ctx, "active_tasks", "Total number of active tasks").Gauge()

	pendingBlocksGauge := metrics.NewCtx(ctx, "pending_block_tasks", "Total number of pending blockstore tasks").Gauge()

	activeBlocksGauge := metrics.NewCtx(ctx, "active_block_tasks", "Total number of active blockstore tasks").Gauge()

	px := process.WithTeardown(func() error {
		return nil
	})

	// onDontHaveTimeout is called when a want-block is sent to a peer that
	// has an old version of Bitswap that doesn't support DONT_HAVE messages,
	// or when no response is received within a timeout.
	var sm *bssm.SessionManager
	var bs *Bitswap
	onDontHaveTimeout := func(p peer.ID, dontHaves []cid.Cid) {
		// Simulate a message arriving with DONT_HAVEs
		if bs.simulateDontHavesOnTimeout {
			sm.ReceiveFrom(ctx, p, nil, nil, dontHaves)
		}
	}
	peerQueueFactory := func(ctx context.Context, p peer.ID) bspm.PeerQueue {
		return bsmq.New(ctx, p, network, onDontHaveTimeout)
	}

	sim := bssim.New()
	bpm := bsbpm.New()
	pm := bspm.New(ctx, peerQueueFactory, network.Self())
	pqm := bspqm.New(ctx, network)

	sessionFactory := func(
		sessctx context.Context,
		sessmgr bssession.SessionManager,
		id uint64,
		spm bssession.SessionPeerManager,
		sim *bssim.SessionInterestManager,
		pm bssession.PeerManager,
		bpm *bsbpm.BlockPresenceManager,
		notif notifications.PubSub,
		provSearchDelay time.Duration,
		rebroadcastDelay delay.D,
		self peer.ID) bssm.Session {
		return bssession.New(sessctx, sessmgr, id, spm, pqm, sim, pm, bpm, notif, provSearchDelay, rebroadcastDelay, self)
	}
	sessionPeerManagerFactory := func(ctx context.Context, id uint64) bssession.SessionPeerManager {
		return bsspm.New(id, network.ConnectionManager())
	}
	notif := notifications.New()
	sm = bssm.New(ctx, sessionFactory, sim, sessionPeerManagerFactory, bpm, pm, notif, network.Self())

	bs = &Bitswap{
		blockstore:                       bstore,
		network:                          network,
		process:                          px,
		newBlocks:                        make(chan cid.Cid, HasBlockBufferSize),
		provideKeys:                      make(chan cid.Cid, provideKeysBufferSize),
		pm:                               pm,
		pqm:                              pqm,
		sm:                               sm,
		sim:                              sim,
		notif:                            notif,
		counters:                         new(counters),
		dupMetric:                        dupHist,
		allMetric:                        allHist,
		sentHistogram:                    sentHistogram,
		sendTimeHistogram:                sendTimeHistogram,
		provideEnabled:                   true,
		provSearchDelay:                  defaults.ProvSearchDelay,
		rebroadcastDelay:                 delay.Fixed(time.Minute),
		engineBstoreWorkerCount:          defaults.BitswapEngineBlockstoreWorkerCount,
		engineTaskWorkerCount:            defaults.BitswapEngineTaskWorkerCount,
		taskWorkerCount:                  defaults.BitswapTaskWorkerCount,
		engineMaxOutstandingBytesPerPeer: defaults.BitswapMaxOutstandingBytesPerPeer,
		engineSetSendDontHaves:           true,
		simulateDontHavesOnTimeout:       true,
	}

	// apply functional options before starting and running bitswap
	for _, option := range options {
		option(bs)
	}

	// Set up decision engine
	bs.engine = decision.NewEngine(
		ctx,
		bstore,
		bs.engineBstoreWorkerCount,
		bs.engineTaskWorkerCount,
		bs.engineMaxOutstandingBytesPerPeer,
		network.ConnectionManager(),
		network.Self(),
		bs.engineScoreLedger,
		pendingEngineGauge,
		activeEngineGauge,
		pendingBlocksGauge,
		activeBlocksGauge,
		decision.WithTaskComparator(bs.taskComparator),
	)
	bs.engine.SetSendDontHaves(bs.engineSetSendDontHaves)

	bs.pqm.Startup()
	network.SetDelegate(bs)

	// Start up bitswaps async worker routines
	bs.startWorkers(ctx, px)
	bs.engine.StartWorkers(ctx, px)

	// bind the context and process.
	// do it over here to avoid closing before all setup is done.
	go func() {
		<-px.Closing() // process closes first
		sm.Shutdown()
		cancelFunc()
		notif.Shutdown()
	}()
	procctx.CloseAfterContext(px, ctx) // parent cancelled first

	return bs
}

// Bitswap instances implement the bitswap protocol.
type Bitswap struct {
	pm *bspm.PeerManager

	// the provider query manager manages requests to find providers
	pqm *bspqm.ProviderQueryManager

	// the engine is the bit of logic that decides who to send which blocks to
	engine *decision.Engine

	// network delivers messages on behalf of the session
	network bsnet.BitSwapNetwork

	// blockstore is the local database
	// NB: ensure threadsafety
	blockstore blockstore.Blockstore

	// manages channels of outgoing blocks for sessions
	notif notifications.PubSub

	// newBlocks is a channel for newly added blocks to be provided to the
	// network.  blocks pushed down this channel get buffered and fed to the
	// provideKeys channel later on to avoid too much network activity
	newBlocks chan cid.Cid
	// provideKeys directly feeds provide workers
	provideKeys chan cid.Cid

	process process.Process

	// Counters for various statistics
	counterLk sync.Mutex
	counters  *counters

	// Metrics interface metrics
	dupMetric         metrics.Histogram
	allMetric         metrics.Histogram
	sentHistogram     metrics.Histogram
	sendTimeHistogram metrics.Histogram

	// External statistics interface
	tracer Tracer

	// the SessionManager routes requests to interested sessions
	sm *bssm.SessionManager

	// the SessionInterestManager keeps track of which sessions are interested
	// in which CIDs
	sim *bssim.SessionInterestManager

	// whether or not to make provide announcements
	provideEnabled bool

	// how long to wait before looking for providers in a session
	provSearchDelay time.Duration

	// how often to rebroadcast providing requests to find more optimized providers
	rebroadcastDelay delay.D

	// how many worker threads to start for decision engine blockstore worker
	engineBstoreWorkerCount int

	// how many worker threads to start for decision engine task worker
	engineTaskWorkerCount int

	// the total number of simultaneous threads sending outgoing messages
	taskWorkerCount int

	// the total amount of bytes that a peer should have outstanding, it is utilized by the decision engine
	engineMaxOutstandingBytesPerPeer int

	// the score ledger used by the decision engine
	engineScoreLedger deciface.ScoreLedger

	// indicates what to do when the engine receives a want-block for a block that
	// is not in the blockstore. Either send DONT_HAVE or do nothing.
	// This is used to simulate older versions of bitswap that did nothing instead of sending back a DONT_HAVE.
	engineSetSendDontHaves bool

	// whether we should actually simulate dont haves on request timeout
	simulateDontHavesOnTimeout bool

	taskComparator TaskComparator
}

type counters struct {
	blocksRecvd    uint64
	dupBlocksRecvd uint64
	dupDataRecvd   uint64
	blocksSent     uint64
	dataSent       uint64
	dataRecvd      uint64
	messagesRecvd  uint64
}

// GetBlock attempts to retrieve a particular block from peers within the
// deadline enforced by the context.
func (bs *Bitswap) GetBlock(parent context.Context, k cid.Cid) (blocks.Block, error) {
	return bsgetter.SyncGetBlock(parent, k, bs.GetBlocks)
}

// WantlistForPeer returns the currently understood list of blocks requested by a
// given peer.
func (bs *Bitswap) WantlistForPeer(p peer.ID) []cid.Cid {
	var out []cid.Cid
	for _, e := range bs.engine.WantlistForPeer(p) {
		out = append(out, e.Cid)
	}
	return out
}

// LedgerForPeer returns aggregated data about blocks swapped and communication
// with a given peer.
func (bs *Bitswap) LedgerForPeer(p peer.ID) *decision.Receipt {
	return bs.engine.LedgerForPeer(p)
}

// GetBlocks returns a channel where the caller may receive blocks that
// correspond to the provided |keys|. Returns an error if BitSwap is unable to
// begin this request within the deadline enforced by the context.
//
// NB: Your request remains open until the context expires. To conserve
// resources, provide a context with a reasonably short deadline (ie. not one
// that lasts throughout the lifetime of the server)
func (bs *Bitswap) GetBlocks(ctx context.Context, keys []cid.Cid) (<-chan blocks.Block, error) {
	session := bs.sm.NewSession(ctx, bs.provSearchDelay, bs.rebroadcastDelay)
	return session.GetBlocks(ctx, keys)
}

// HasBlock announces the existence of a block to this bitswap service. The
// service will potentially notify its peers.
func (bs *Bitswap) HasBlock(ctx context.Context, blk blocks.Block) error {
	return bs.receiveBlocksFrom(ctx, "", []blocks.Block{blk}, nil, nil)
}

// TODO: Some of this stuff really only needs to be done when adding a block
// from the user, not when receiving it from the network.
// In case you run `git blame` on this comment, I'll save you some time: ask
// @whyrusleeping, I don't know the answers you seek.
func (bs *Bitswap) receiveBlocksFrom(ctx context.Context, from peer.ID, blks []blocks.Block, haves []cid.Cid, dontHaves []cid.Cid) error {
	select {
	case <-bs.process.Closing():
		return errors.New("bitswap is closed")
	default:
	}

	wanted := blks

	// If blocks came from the network
	if from != "" {
		var notWanted []blocks.Block
		wanted, notWanted = bs.sim.SplitWantedUnwanted(blks)
		for _, b := range notWanted {
			log.Debugf("[recv] block not in wantlist; cid=%s, peer=%s", b.Cid(), from)
		}
	}

	// Put wanted blocks into blockstore
	if len(wanted) > 0 {
		err := bs.blockstore.PutMany(ctx, wanted)
		if err != nil {
			log.Errorf("Error writing %d blocks to datastore: %s", len(wanted), err)
			return err
		}
	}

	// NOTE: There exists the possiblity for a race condition here.  If a user
	// creates a node, then adds it to the dagservice while another goroutine
	// is waiting on a GetBlock for that object, they will receive a reference
	// to the same node. We should address this soon, but i'm not going to do
	// it now as it requires more thought and isnt causing immediate problems.

	allKs := make([]cid.Cid, 0, len(blks))
	for _, b := range blks {
		allKs = append(allKs, b.Cid())
	}

	// If the message came from the network
	if from != "" {
		// Inform the PeerManager so that we can calculate per-peer latency
		combined := make([]cid.Cid, 0, len(allKs)+len(haves)+len(dontHaves))
		combined = append(combined, allKs...)
		combined = append(combined, haves...)
		combined = append(combined, dontHaves...)
		bs.pm.ResponseReceived(from, combined)
	}

	// Send all block keys (including duplicates) to any sessions that want them.
	// (The duplicates are needed by sessions for accounting purposes)
	bs.sm.ReceiveFrom(ctx, from, allKs, haves, dontHaves)

	// Send wanted blocks to decision engine
	bs.engine.ReceiveFrom(from, wanted)

	// Publish the block to any Bitswap clients that had requested blocks.
	// (the sessions use this pubsub mechanism to inform clients of incoming
	// blocks)
	for _, b := range wanted {
		bs.notif.Publish(b)
	}

	// If the reprovider is enabled, send wanted blocks to reprovider
	if bs.provideEnabled {
		for _, blk := range wanted {
			select {
			case bs.newBlocks <- blk.Cid():
				// send block off to be reprovided
			case <-bs.process.Closing():
				return bs.process.Close()
			}
		}
	}

	if from != "" {
		for _, b := range wanted {
			log.Debugw("Bitswap.GetBlockRequest.End", "cid", b.Cid())
		}
	}

	return nil
}

// ReceiveMessage is called by the network interface when a new message is
// received.
func (bs *Bitswap) ReceiveMessage(ctx context.Context, p peer.ID, incoming bsmsg.BitSwapMessage) {
	bs.counterLk.Lock()
	bs.counters.messagesRecvd++
	bs.counterLk.Unlock()

	// This call records changes to wantlists, blocks received,
	// and number of bytes transfered.
	bs.engine.MessageReceived(ctx, p, incoming)
	// TODO: this is bad, and could be easily abused.
	// Should only track *useful* messages in ledger

	if bs.tracer != nil {
		bs.tracer.MessageReceived(p, incoming)
	}

	iblocks := incoming.Blocks()

	if len(iblocks) > 0 {
		bs.updateReceiveCounters(iblocks)
		for _, b := range iblocks {
			log.Debugf("[recv] block; cid=%s, peer=%s", b.Cid(), p)
		}
	}

	haves := incoming.Haves()
	dontHaves := incoming.DontHaves()
	if len(iblocks) > 0 || len(haves) > 0 || len(dontHaves) > 0 {
		// Process blocks
		err := bs.receiveBlocksFrom(ctx, p, iblocks, haves, dontHaves)
		if err != nil {
			log.Warnf("ReceiveMessage recvBlockFrom error: %s", err)
			return
		}
	}
}

func (bs *Bitswap) updateReceiveCounters(blocks []blocks.Block) {
	// Check which blocks are in the datastore
	// (Note: any errors from the blockstore are simply logged out in
	// blockstoreHas())
	blocksHas := bs.blockstoreHas(blocks)

	bs.counterLk.Lock()
	defer bs.counterLk.Unlock()

	// Do some accounting for each block
	for i, b := range blocks {
		has := blocksHas[i]

		blkLen := len(b.RawData())
		bs.allMetric.Observe(float64(blkLen))
		if has {
			bs.dupMetric.Observe(float64(blkLen))
		}

		c := bs.counters

		c.blocksRecvd++
		c.dataRecvd += uint64(blkLen)
		if has {
			c.dupBlocksRecvd++
			c.dupDataRecvd += uint64(blkLen)
		}
	}
}

func (bs *Bitswap) blockstoreHas(blks []blocks.Block) []bool {
	res := make([]bool, len(blks))

	wg := sync.WaitGroup{}
	for i, block := range blks {
		wg.Add(1)
		go func(i int, b blocks.Block) {
			defer wg.Done()

			has, err := bs.blockstore.Has(context.TODO(), b.Cid())
			if err != nil {
				log.Infof("blockstore.Has error: %s", err)
				has = false
			}

			res[i] = has
		}(i, block)
	}
	wg.Wait()

	return res
}

// PeerConnected is called by the network interface
// when a peer initiates a new connection to bitswap.
func (bs *Bitswap) PeerConnected(p peer.ID) {
	bs.pm.Connected(p)
	bs.engine.PeerConnected(p)
}

// PeerDisconnected is called by the network interface when a peer
// closes a connection
func (bs *Bitswap) PeerDisconnected(p peer.ID) {
	bs.pm.Disconnected(p)
	bs.engine.PeerDisconnected(p)
}

// ReceiveError is called by the network interface when an error happens
// at the network layer. Currently just logs error.
func (bs *Bitswap) ReceiveError(err error) {
	log.Infof("Bitswap ReceiveError: %s", err)
	// TODO log the network error
	// TODO bubble the network error up to the parent context/error logger
}

// Close is called to shutdown Bitswap
func (bs *Bitswap) Close() error {
	return bs.process.Close()
}

// GetWantlist returns the current local wantlist (both want-blocks and
// want-haves).
func (bs *Bitswap) GetWantlist() []cid.Cid {
	return bs.pm.CurrentWants()
}

// GetWantBlocks returns the current list of want-blocks.
func (bs *Bitswap) GetWantBlocks() []cid.Cid {
	return bs.pm.CurrentWantBlocks()
}

// GetWanthaves returns the current list of want-haves.
func (bs *Bitswap) GetWantHaves() []cid.Cid {
	return bs.pm.CurrentWantHaves()
}

// IsOnline is needed to match go-ipfs-exchange-interface
func (bs *Bitswap) IsOnline() bool {
	return true
}

// NewSession generates a new Bitswap session. You should use this, rather
// that calling Bitswap.GetBlocks, any time you intend to do several related
// block requests in a row. The session returned will have it's own GetBlocks
// method, but the session will use the fact that the requests are related to
// be more efficient in its requests to peers. If you are using a session
// from go-blockservice, it will create a bitswap session automatically.
func (bs *Bitswap) NewSession(ctx context.Context) exchange.Fetcher {
	return bs.sm.NewSession(ctx, bs.provSearchDelay, bs.rebroadcastDelay)
}
