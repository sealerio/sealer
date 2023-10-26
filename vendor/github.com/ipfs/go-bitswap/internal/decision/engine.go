// Package decision implements the decision engine for the bitswap service.
package decision

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	bsmsg "github.com/ipfs/go-bitswap/message"
	pb "github.com/ipfs/go-bitswap/message/pb"
	wl "github.com/ipfs/go-bitswap/wantlist"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-metrics-interface"
	"github.com/ipfs/go-peertaskqueue"
	"github.com/ipfs/go-peertaskqueue/peertask"
	"github.com/ipfs/go-peertaskqueue/peertracker"
	process "github.com/jbenet/goprocess"
	"github.com/libp2p/go-libp2p-core/peer"
)

// TODO consider taking responsibility for other types of requests. For
// example, there could be a |cancelQueue| for all of the cancellation
// messages that need to go out. There could also be a |wantlistQueue| for
// the local peer's wantlists. Alternatively, these could all be bundled
// into a single, intelligent global queue that efficiently
// batches/combines and takes all of these into consideration.
//
// Right now, messages go onto the network for four reasons:
// 1. an initial `sendwantlist` message to a provider of the first key in a
//    request
// 2. a periodic full sweep of `sendwantlist` messages to all providers
// 3. upon receipt of blocks, a `cancel` message to all peers
// 4. draining the priority queue of `blockrequests` from peers
//
// Presently, only `blockrequests` are handled by the decision engine.
// However, there is an opportunity to give it more responsibility! If the
// decision engine is given responsibility for all of the others, it can
// intelligently decide how to combine requests efficiently.
//
// Some examples of what would be possible:
//
// * when sending out the wantlists, include `cancel` requests
// * when handling `blockrequests`, include `sendwantlist` and `cancel` as
//   appropriate
// * when handling `cancel`, if we recently received a wanted block from a
//   peer, include a partial wantlist that contains a few other high priority
//   blocks
//
// In a sense, if we treat the decision engine as a black box, it could do
// whatever it sees fit to produce desired outcomes (get wanted keys
// quickly, maintain good relationships with peers, etc).

var log = logging.Logger("engine")

const (
	// outboxChanBuffer must be 0 to prevent stale messages from being sent
	outboxChanBuffer = 0
	// targetMessageSize is the ideal size of the batched payload. We try to
	// pop this much data off the request queue, but it may be a little more
	// or less depending on what's in the queue.
	targetMessageSize = 16 * 1024
	// tagFormat is the tag given to peers associated an engine
	tagFormat = "bs-engine-%s-%s"

	// queuedTagWeight is the default weight for peers that have work queued
	// on their behalf.
	queuedTagWeight = 10

	// maxBlockSizeReplaceHasWithBlock is the maximum size of the block in
	// bytes up to which we will replace a want-have with a want-block
	maxBlockSizeReplaceHasWithBlock = 1024
)

// Envelope contains a message for a Peer.
type Envelope struct {
	// Peer is the intended recipient.
	Peer peer.ID

	// Message is the payload.
	Message bsmsg.BitSwapMessage

	// A callback to notify the decision queue that the task is complete
	Sent func()
}

// PeerTagger covers the methods on the connection manager used by the decision
// engine to tag peers
type PeerTagger interface {
	TagPeer(peer.ID, string, int)
	UntagPeer(p peer.ID, tag string)
}

// Assigns a specific score to a peer
type ScorePeerFunc func(peer.ID, int)

// ScoreLedger is an external ledger dealing with peer scores.
type ScoreLedger interface {
	// Returns aggregated data communication with a given peer.
	GetReceipt(p peer.ID) *Receipt
	// Increments the sent counter for the given peer.
	AddToSentBytes(p peer.ID, n int)
	// Increments the received counter for the given peer.
	AddToReceivedBytes(p peer.ID, n int)
	// PeerConnected should be called when a new peer connects,
	// meaning the ledger should open accounting.
	PeerConnected(p peer.ID)
	// PeerDisconnected should be called when a peer disconnects to
	// clean up the accounting.
	PeerDisconnected(p peer.ID)
	// Starts the ledger sampling process.
	Start(scorePeer ScorePeerFunc)
	// Stops the sampling process.
	Stop()
}

// Engine manages sending requested blocks to peers.
type Engine struct {
	// peerRequestQueue is a priority queue of requests received from peers.
	// Requests are popped from the queue, packaged up, and placed in the
	// outbox.
	peerRequestQueue *peertaskqueue.PeerTaskQueue

	// FIXME it's a bit odd for the client and the worker to both share memory
	// (both modify the peerRequestQueue) and also to communicate over the
	// workSignal channel. consider sending requests over the channel and
	// allowing the worker to have exclusive access to the peerRequestQueue. In
	// that case, no lock would be required.
	workSignal chan struct{}

	// outbox contains outgoing messages to peers. This is owned by the
	// taskWorker goroutine
	outbox chan (<-chan *Envelope)

	bsm *blockstoreManager

	peerTagger PeerTagger

	tagQueued, tagUseful string

	lock sync.RWMutex // protects the fields immediately below

	// ledgerMap lists block-related Ledgers by their Partner key.
	ledgerMap map[peer.ID]*ledger

	// peerLedger saves which peers are waiting for a Cid
	peerLedger *peerLedger

	// an external ledger dealing with peer scores
	scoreLedger ScoreLedger

	ticker *time.Ticker

	taskWorkerLock  sync.Mutex
	taskWorkerCount int

	// maxBlockSizeReplaceHasWithBlock is the maximum size of the block in
	// bytes up to which we will replace a want-have with a want-block
	maxBlockSizeReplaceHasWithBlock int

	sendDontHaves bool

	self peer.ID

	// metrics gauge for total pending tasks across all workers
	pendingGauge metrics.Gauge

	// metrics gauge for total pending tasks across all workers
	activeGauge metrics.Gauge

	// used to ensure metrics are reported each fixed number of operation
	metricsLock         sync.Mutex
	metricUpdateCounter int

	taskComparator TaskComparator
}

// TaskInfo represents the details of a request from a peer.
type TaskInfo struct {
	Peer peer.ID
	// The CID of the block
	Cid cid.Cid
	// Tasks can be want-have or want-block
	IsWantBlock bool
	// Whether to immediately send a response if the block is not found
	SendDontHave bool
	// The size of the block corresponding to the task
	BlockSize int
	// Whether the block was found
	HaveBlock bool
}

// TaskComparator is used for task prioritization.
// It should return true if task 'ta' has higher priority than task 'tb'
type TaskComparator func(ta, tb *TaskInfo) bool

type Option func(*Engine)

func WithTaskComparator(comparator TaskComparator) Option {
	return func(e *Engine) {
		e.taskComparator = comparator
	}
}

// wrapTaskComparator wraps a TaskComparator so it can be used as a QueueTaskComparator
func wrapTaskComparator(tc TaskComparator) peertask.QueueTaskComparator {
	return func(a, b *peertask.QueueTask) bool {
		taskDataA := a.Task.Data.(*taskData)
		taskInfoA := &TaskInfo{
			Peer:         a.Target,
			Cid:          a.Task.Topic.(cid.Cid),
			IsWantBlock:  taskDataA.IsWantBlock,
			SendDontHave: taskDataA.SendDontHave,
			BlockSize:    taskDataA.BlockSize,
			HaveBlock:    taskDataA.HaveBlock,
		}
		taskDataB := b.Task.Data.(*taskData)
		taskInfoB := &TaskInfo{
			Peer:         b.Target,
			Cid:          b.Task.Topic.(cid.Cid),
			IsWantBlock:  taskDataB.IsWantBlock,
			SendDontHave: taskDataB.SendDontHave,
			BlockSize:    taskDataB.BlockSize,
			HaveBlock:    taskDataB.HaveBlock,
		}
		return tc(taskInfoA, taskInfoB)
	}
}

// NewEngine creates a new block sending engine for the given block store.
// maxOutstandingBytesPerPeer hints to the peer task queue not to give a peer more tasks if it has some maximum
// work already outstanding.
func NewEngine(
	ctx context.Context,
	bs bstore.Blockstore,
	bstoreWorkerCount,
	engineTaskWorkerCount, maxOutstandingBytesPerPeer int,
	peerTagger PeerTagger,
	self peer.ID,
	scoreLedger ScoreLedger,
	pendingEngineGauge metrics.Gauge,
	activeEngineGauge metrics.Gauge,
	pendingBlocksGauge metrics.Gauge,
	activeBlocksGauge metrics.Gauge,
	opts ...Option,
) *Engine {
	return newEngine(
		ctx,
		bs,
		bstoreWorkerCount,
		engineTaskWorkerCount,
		maxOutstandingBytesPerPeer,
		peerTagger,
		self,
		maxBlockSizeReplaceHasWithBlock,
		scoreLedger,
		pendingEngineGauge,
		activeEngineGauge,
		pendingBlocksGauge,
		activeBlocksGauge,
		opts...,
	)
}

func newEngine(
	ctx context.Context,
	bs bstore.Blockstore,
	bstoreWorkerCount,
	engineTaskWorkerCount, maxOutstandingBytesPerPeer int,
	peerTagger PeerTagger,
	self peer.ID,
	maxReplaceSize int,
	scoreLedger ScoreLedger,
	pendingEngineGauge metrics.Gauge,
	activeEngineGauge metrics.Gauge,
	pendingBlocksGauge metrics.Gauge,
	activeBlocksGauge metrics.Gauge,
	opts ...Option,
) *Engine {

	if scoreLedger == nil {
		scoreLedger = NewDefaultScoreLedger()
	}

	e := &Engine{
		ledgerMap:                       make(map[peer.ID]*ledger),
		scoreLedger:                     scoreLedger,
		bsm:                             newBlockstoreManager(ctx, bs, bstoreWorkerCount, pendingBlocksGauge, activeBlocksGauge),
		peerTagger:                      peerTagger,
		outbox:                          make(chan (<-chan *Envelope), outboxChanBuffer),
		workSignal:                      make(chan struct{}, 1),
		ticker:                          time.NewTicker(time.Millisecond * 100),
		maxBlockSizeReplaceHasWithBlock: maxReplaceSize,
		taskWorkerCount:                 engineTaskWorkerCount,
		sendDontHaves:                   true,
		self:                            self,
		peerLedger:                      newPeerLedger(),
		pendingGauge:                    pendingEngineGauge,
		activeGauge:                     activeEngineGauge,
	}
	e.tagQueued = fmt.Sprintf(tagFormat, "queued", uuid.New().String())
	e.tagUseful = fmt.Sprintf(tagFormat, "useful", uuid.New().String())

	for _, opt := range opts {
		opt(e)
	}

	// default peer task queue options
	peerTaskQueueOpts := []peertaskqueue.Option{
		peertaskqueue.OnPeerAddedHook(e.onPeerAdded),
		peertaskqueue.OnPeerRemovedHook(e.onPeerRemoved),
		peertaskqueue.TaskMerger(newTaskMerger()),
		peertaskqueue.IgnoreFreezing(true),
		peertaskqueue.MaxOutstandingWorkPerPeer(maxOutstandingBytesPerPeer),
	}

	if e.taskComparator != nil {
		queueTaskComparator := wrapTaskComparator(e.taskComparator)
		peerTaskQueueOpts = append(peerTaskQueueOpts, peertaskqueue.PeerComparator(peertracker.TaskPriorityPeerComparator(queueTaskComparator)))
		peerTaskQueueOpts = append(peerTaskQueueOpts, peertaskqueue.TaskComparator(queueTaskComparator))
	}

	e.peerRequestQueue = peertaskqueue.New(peerTaskQueueOpts...)

	return e
}

func (e *Engine) updateMetrics() {
	e.metricsLock.Lock()
	c := e.metricUpdateCounter
	e.metricUpdateCounter++
	e.metricsLock.Unlock()

	if c%100 == 0 {
		stats := e.peerRequestQueue.Stats()
		e.activeGauge.Set(float64(stats.NumActive))
		e.pendingGauge.Set(float64(stats.NumPending))
	}
}

// SetSendDontHaves indicates what to do when the engine receives a want-block
// for a block that is not in the blockstore. Either
// - Send a DONT_HAVE message
// - Simply don't respond
// Older versions of Bitswap did not respond, so this allows us to simulate
// those older versions for testing.
func (e *Engine) SetSendDontHaves(send bool) {
	e.sendDontHaves = send
}

// Starts the score ledger. Before start the function checks and,
// if it is unset, initializes the scoreLedger with the default
// implementation.
func (e *Engine) startScoreLedger(px process.Process) {
	e.scoreLedger.Start(func(p peer.ID, score int) {
		if score == 0 {
			e.peerTagger.UntagPeer(p, e.tagUseful)
		} else {
			e.peerTagger.TagPeer(p, e.tagUseful, score)
		}
	})
	px.Go(func(ppx process.Process) {
		<-ppx.Closing()
		e.scoreLedger.Stop()
	})
}

// Start up workers to handle requests from other nodes for the data on this node
func (e *Engine) StartWorkers(ctx context.Context, px process.Process) {
	// Start up blockstore manager
	e.bsm.start(px)
	e.startScoreLedger(px)

	e.taskWorkerLock.Lock()
	defer e.taskWorkerLock.Unlock()

	for i := 0; i < e.taskWorkerCount; i++ {
		px.Go(func(px process.Process) {
			e.taskWorker(ctx)
		})
	}
}

func (e *Engine) onPeerAdded(p peer.ID) {
	e.peerTagger.TagPeer(p, e.tagQueued, queuedTagWeight)
}

func (e *Engine) onPeerRemoved(p peer.ID) {
	e.peerTagger.UntagPeer(p, e.tagQueued)
}

// WantlistForPeer returns the list of keys that the given peer has asked for
func (e *Engine) WantlistForPeer(p peer.ID) []wl.Entry {
	partner := e.findOrCreate(p)

	partner.lk.Lock()
	entries := partner.wantList.Entries()
	partner.lk.Unlock()

	return entries
}

// LedgerForPeer returns aggregated data communication with a given peer.
func (e *Engine) LedgerForPeer(p peer.ID) *Receipt {
	return e.scoreLedger.GetReceipt(p)
}

// Each taskWorker pulls items off the request queue up to the maximum size
// and adds them to an envelope that is passed off to the bitswap workers,
// which send the message to the network.
func (e *Engine) taskWorker(ctx context.Context) {
	defer e.taskWorkerExit()
	for {
		oneTimeUse := make(chan *Envelope, 1) // buffer to prevent blocking
		select {
		case <-ctx.Done():
			return
		case e.outbox <- oneTimeUse:
		}
		// receiver is ready for an outoing envelope. let's prepare one. first,
		// we must acquire a task from the PQ...
		envelope, err := e.nextEnvelope(ctx)
		if err != nil {
			close(oneTimeUse)
			return // ctx cancelled
		}
		oneTimeUse <- envelope // buffered. won't block
		close(oneTimeUse)
	}
}

// taskWorkerExit handles cleanup of task workers
func (e *Engine) taskWorkerExit() {
	e.taskWorkerLock.Lock()
	defer e.taskWorkerLock.Unlock()

	e.taskWorkerCount--
	if e.taskWorkerCount == 0 {
		close(e.outbox)
	}
}

// nextEnvelope runs in the taskWorker goroutine. Returns an error if the
// context is cancelled before the next Envelope can be created.
func (e *Engine) nextEnvelope(ctx context.Context) (*Envelope, error) {
	for {
		// Pop some tasks off the request queue
		p, nextTasks, pendingBytes := e.peerRequestQueue.PopTasks(targetMessageSize)
		e.updateMetrics()
		for len(nextTasks) == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-e.workSignal:
				p, nextTasks, pendingBytes = e.peerRequestQueue.PopTasks(targetMessageSize)
				e.updateMetrics()
			case <-e.ticker.C:
				// When a task is cancelled, the queue may be "frozen" for a
				// period of time. We periodically "thaw" the queue to make
				// sure it doesn't get stuck in a frozen state.
				e.peerRequestQueue.ThawRound()
				p, nextTasks, pendingBytes = e.peerRequestQueue.PopTasks(targetMessageSize)
				e.updateMetrics()
			}
		}

		// Create a new message
		msg := bsmsg.New(false)

		log.Debugw("Bitswap process tasks", "local", e.self, "taskCount", len(nextTasks))

		// Amount of data in the request queue still waiting to be popped
		msg.SetPendingBytes(int32(pendingBytes))

		// Split out want-blocks, want-haves and DONT_HAVEs
		blockCids := make([]cid.Cid, 0, len(nextTasks))
		blockTasks := make(map[cid.Cid]*taskData, len(nextTasks))
		for _, t := range nextTasks {
			c := t.Topic.(cid.Cid)
			td := t.Data.(*taskData)
			if td.HaveBlock {
				if td.IsWantBlock {
					blockCids = append(blockCids, c)
					blockTasks[c] = td
				} else {
					// Add HAVES to the message
					msg.AddHave(c)
				}
			} else {
				// Add DONT_HAVEs to the message
				msg.AddDontHave(c)
			}
		}

		// Fetch blocks from datastore
		blks, err := e.bsm.getBlocks(ctx, blockCids)
		if err != nil {
			// we're dropping the envelope but that's not an issue in practice.
			return nil, err
		}

		for c, t := range blockTasks {
			blk := blks[c]
			// If the block was not found (it has been removed)
			if blk == nil {
				// If the client requested DONT_HAVE, add DONT_HAVE to the message
				if t.SendDontHave {
					msg.AddDontHave(c)
				}
			} else {
				// Add the block to the message
				// log.Debugf("  make evlp %s->%s block: %s (%d bytes)", e.self, p, c, len(blk.RawData()))
				msg.AddBlock(blk)
			}
		}

		// If there's nothing in the message, bail out
		if msg.Empty() {
			e.peerRequestQueue.TasksDone(p, nextTasks...)
			continue
		}

		log.Debugw("Bitswap engine -> msg", "local", e.self, "to", p, "blockCount", len(msg.Blocks()), "presenceCount", len(msg.BlockPresences()), "size", msg.Size())
		return &Envelope{
			Peer:    p,
			Message: msg,
			Sent: func() {
				// Once the message has been sent, signal the request queue so
				// it can be cleared from the queue
				e.peerRequestQueue.TasksDone(p, nextTasks...)

				// Signal the worker to check for more work
				e.signalNewWork()
			},
		}, nil
	}
}

// Outbox returns a channel of one-time use Envelope channels.
func (e *Engine) Outbox() <-chan (<-chan *Envelope) {
	return e.outbox
}

// Peers returns a slice of Peers with whom the local node has active sessions.
func (e *Engine) Peers() []peer.ID {
	e.lock.RLock()
	defer e.lock.RUnlock()

	response := make([]peer.ID, 0, len(e.ledgerMap))

	for _, ledger := range e.ledgerMap {
		response = append(response, ledger.Partner)
	}
	return response
}

// MessageReceived is called when a message is received from a remote peer.
// For each item in the wantlist, add a want-have or want-block entry to the
// request queue (this is later popped off by the workerTasks)
func (e *Engine) MessageReceived(ctx context.Context, p peer.ID, m bsmsg.BitSwapMessage) {
	entries := m.Wantlist()

	if len(entries) > 0 {
		log.Debugw("Bitswap engine <- msg", "local", e.self, "from", p, "entryCount", len(entries))
		for _, et := range entries {
			if !et.Cancel {
				if et.WantType == pb.Message_Wantlist_Have {
					log.Debugw("Bitswap engine <- want-have", "local", e.self, "from", p, "cid", et.Cid)
				} else {
					log.Debugw("Bitswap engine <- want-block", "local", e.self, "from", p, "cid", et.Cid)
				}
			}
		}
	}

	if m.Empty() {
		log.Infof("received empty message from %s", p)
	}

	newWorkExists := false
	defer func() {
		if newWorkExists {
			e.signalNewWork()
		}
	}()

	// Get block sizes
	wants, cancels := e.splitWantsCancels(entries)
	wantKs := cid.NewSet()
	for _, entry := range wants {
		wantKs.Add(entry.Cid)
	}
	blockSizes, err := e.bsm.getBlockSizes(ctx, wantKs.Keys())
	if err != nil {
		log.Info("aborting message processing", err)
		return
	}

	e.lock.Lock()
	for _, entry := range wants {
		e.peerLedger.Wants(p, entry.Cid)
	}
	for _, entry := range cancels {
		e.peerLedger.CancelWant(p, entry.Cid)
	}
	e.lock.Unlock()

	// Get the ledger for the peer
	l := e.findOrCreate(p)
	l.lk.Lock()
	defer l.lk.Unlock()

	// If the peer sent a full wantlist, replace the ledger's wantlist
	if m.Full() {
		l.wantList = wl.New()
	}

	var activeEntries []peertask.Task

	// Remove cancelled blocks from the queue
	for _, entry := range cancels {
		log.Debugw("Bitswap engine <- cancel", "local", e.self, "from", p, "cid", entry.Cid)
		if l.CancelWant(entry.Cid) {
			e.peerRequestQueue.Remove(entry.Cid, p)
		}
	}

	// For each want-have / want-block
	for _, entry := range wants {
		c := entry.Cid
		blockSize, found := blockSizes[entry.Cid]

		// Add each want-have / want-block to the ledger
		l.Wants(c, entry.Priority, entry.WantType)

		// If the block was not found
		if !found {
			log.Debugw("Bitswap engine: block not found", "local", e.self, "from", p, "cid", entry.Cid, "sendDontHave", entry.SendDontHave)

			// Only add the task to the queue if the requester wants a DONT_HAVE
			if e.sendDontHaves && entry.SendDontHave {
				newWorkExists = true
				isWantBlock := false
				if entry.WantType == pb.Message_Wantlist_Block {
					isWantBlock = true
				}

				activeEntries = append(activeEntries, peertask.Task{
					Topic:    c,
					Priority: int(entry.Priority),
					Work:     bsmsg.BlockPresenceSize(c),
					Data: &taskData{
						BlockSize:    0,
						HaveBlock:    false,
						IsWantBlock:  isWantBlock,
						SendDontHave: entry.SendDontHave,
					},
				})
			}
		} else {
			// The block was found, add it to the queue
			newWorkExists = true

			isWantBlock := e.sendAsBlock(entry.WantType, blockSize)

			log.Debugw("Bitswap engine: block found", "local", e.self, "from", p, "cid", entry.Cid, "isWantBlock", isWantBlock)

			// entrySize is the amount of space the entry takes up in the
			// message we send to the recipient. If we're sending a block, the
			// entrySize is the size of the block. Otherwise it's the size of
			// a block presence entry.
			entrySize := blockSize
			if !isWantBlock {
				entrySize = bsmsg.BlockPresenceSize(c)
			}
			activeEntries = append(activeEntries, peertask.Task{
				Topic:    c,
				Priority: int(entry.Priority),
				Work:     entrySize,
				Data: &taskData{
					BlockSize:    blockSize,
					HaveBlock:    true,
					IsWantBlock:  isWantBlock,
					SendDontHave: entry.SendDontHave,
				},
			})
		}
	}

	// Push entries onto the request queue
	if len(activeEntries) > 0 {
		e.peerRequestQueue.PushTasks(p, activeEntries...)
		e.updateMetrics()
	}
}

// Split the want-have / want-block entries from the cancel entries
func (e *Engine) splitWantsCancels(es []bsmsg.Entry) ([]bsmsg.Entry, []bsmsg.Entry) {
	wants := make([]bsmsg.Entry, 0, len(es))
	cancels := make([]bsmsg.Entry, 0, len(es))
	for _, et := range es {
		if et.Cancel {
			cancels = append(cancels, et)
		} else {
			wants = append(wants, et)
		}
	}
	return wants, cancels
}

// ReceiveFrom is called when new blocks are received and added to the block
// store, meaning there may be peers who want those blocks, so we should send
// the blocks to them.
//
// This function also updates the receive side of the ledger.
func (e *Engine) ReceiveFrom(from peer.ID, blks []blocks.Block) {
	if len(blks) == 0 {
		return
	}

	if from != "" {
		l := e.findOrCreate(from)
		l.lk.Lock()

		// Record how many bytes were received in the ledger
		for _, blk := range blks {
			log.Debugw("Bitswap engine <- block", "local", e.self, "from", from, "cid", blk.Cid(), "size", len(blk.RawData()))
			e.scoreLedger.AddToReceivedBytes(l.Partner, len(blk.RawData()))
		}

		l.lk.Unlock()
	}

	// Get the size of each block
	blockSizes := make(map[cid.Cid]int, len(blks))
	for _, blk := range blks {
		blockSizes[blk.Cid()] = len(blk.RawData())
	}

	// Check each peer to see if it wants one of the blocks we received
	var work bool
	missingWants := make(map[peer.ID][]cid.Cid)
	for _, b := range blks {
		k := b.Cid()

		e.lock.RLock()
		peers := e.peerLedger.Peers(k)
		e.lock.RUnlock()

		for _, p := range peers {
			e.lock.RLock()
			ledger, ok := e.ledgerMap[p]
			e.lock.RUnlock()

			if !ok {
				// This can happen if the peer has disconnected while we're processing this list.
				log.Debugw("failed to find peer in ledger", "peer", p)
				missingWants[p] = append(missingWants[p], k)
				continue
			}
			ledger.lk.RLock()
			entry, ok := ledger.WantListContains(k)
			ledger.lk.RUnlock()
			if !ok {
				// This can happen if the peer has canceled their want while we're processing this message.
				log.Debugw("wantlist index doesn't match peer's wantlist", "peer", p)
				missingWants[p] = append(missingWants[p], k)
				continue
			}
			work = true

			blockSize := blockSizes[k]
			isWantBlock := e.sendAsBlock(entry.WantType, blockSize)

			entrySize := blockSize
			if !isWantBlock {
				entrySize = bsmsg.BlockPresenceSize(k)
			}

			e.peerRequestQueue.PushTasks(p, peertask.Task{
				Topic:    entry.Cid,
				Priority: int(entry.Priority),
				Work:     entrySize,
				Data: &taskData{
					BlockSize:    blockSize,
					HaveBlock:    true,
					IsWantBlock:  isWantBlock,
					SendDontHave: false,
				},
			})
			e.updateMetrics()
		}
	}

	// If we found missing wants (e.g., because the peer disconnected, we have some races here)
	// remove them from the list. Unfortunately, we still have to re-check because the user
	// could have re-connected in the meantime.
	if len(missingWants) > 0 {
		e.lock.Lock()
		for p, wl := range missingWants {
			if ledger, ok := e.ledgerMap[p]; ok {
				ledger.lk.RLock()
				for _, k := range wl {
					if _, has := ledger.WantListContains(k); has {
						continue
					}
					e.peerLedger.CancelWant(p, k)
				}
				ledger.lk.RUnlock()
			} else {
				for _, k := range wl {
					e.peerLedger.CancelWant(p, k)
				}
			}
		}
		e.lock.Unlock()
	}

	if work {
		e.signalNewWork()
	}
}

// TODO add contents of m.WantList() to my local wantlist? NB: could introduce
// race conditions where I send a message, but MessageSent gets handled after
// MessageReceived. The information in the local wantlist could become
// inconsistent. Would need to ensure that Sends and acknowledgement of the
// send happen atomically

// MessageSent is called when a message has successfully been sent out, to record
// changes.
func (e *Engine) MessageSent(p peer.ID, m bsmsg.BitSwapMessage) {
	l := e.findOrCreate(p)
	l.lk.Lock()
	defer l.lk.Unlock()

	// Remove sent blocks from the want list for the peer
	for _, block := range m.Blocks() {
		e.scoreLedger.AddToSentBytes(l.Partner, len(block.RawData()))
		l.wantList.RemoveType(block.Cid(), pb.Message_Wantlist_Block)
	}

	// Remove sent block presences from the want list for the peer
	for _, bp := range m.BlockPresences() {
		// Don't record sent data. We reserve that for data blocks.
		if bp.Type == pb.Message_Have {
			l.wantList.RemoveType(bp.Cid, pb.Message_Wantlist_Have)
		}
	}
}

// PeerConnected is called when a new peer connects, meaning we should start
// sending blocks.
func (e *Engine) PeerConnected(p peer.ID) {
	e.lock.Lock()
	defer e.lock.Unlock()

	_, ok := e.ledgerMap[p]
	if !ok {
		e.ledgerMap[p] = newLedger(p)
	}

	e.scoreLedger.PeerConnected(p)
}

// PeerDisconnected is called when a peer disconnects.
func (e *Engine) PeerDisconnected(p peer.ID) {
	e.lock.Lock()
	defer e.lock.Unlock()

	ledger, ok := e.ledgerMap[p]
	if ok {
		ledger.lk.RLock()
		entries := ledger.Entries()
		ledger.lk.RUnlock()

		for _, entry := range entries {
			e.peerLedger.CancelWant(p, entry.Cid)
		}
	}
	delete(e.ledgerMap, p)

	e.scoreLedger.PeerDisconnected(p)
}

// If the want is a want-have, and it's below a certain size, send the full
// block (instead of sending a HAVE)
func (e *Engine) sendAsBlock(wantType pb.Message_Wantlist_WantType, blockSize int) bool {
	isWantBlock := wantType == pb.Message_Wantlist_Block
	return isWantBlock || blockSize <= e.maxBlockSizeReplaceHasWithBlock
}

func (e *Engine) numBytesSentTo(p peer.ID) uint64 {
	return e.LedgerForPeer(p).Sent
}

func (e *Engine) numBytesReceivedFrom(p peer.ID) uint64 {
	return e.LedgerForPeer(p).Recv
}

// ledger lazily instantiates a ledger
func (e *Engine) findOrCreate(p peer.ID) *ledger {
	// Take a read lock (as it's less expensive) to check if we have a ledger
	// for the peer
	e.lock.RLock()
	l, ok := e.ledgerMap[p]
	e.lock.RUnlock()
	if ok {
		return l
	}

	// There's no ledger, so take a write lock, then check again and create the
	// ledger if necessary
	e.lock.Lock()
	defer e.lock.Unlock()
	l, ok = e.ledgerMap[p]
	if !ok {
		l = newLedger(p)
		e.ledgerMap[p] = l
	}
	return l
}

func (e *Engine) signalNewWork() {
	// Signal task generation to restart (if stopped!)
	select {
	case e.workSignal <- struct{}{}:
	default:
	}
}
