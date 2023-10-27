package peertaskqueue

import (
	"sync"

	pq "github.com/ipfs/go-ipfs-pq"
	"github.com/ipfs/go-peertaskqueue/peertask"
	"github.com/ipfs/go-peertaskqueue/peertracker"
	"github.com/libp2p/go-libp2p-core/peer"
)

type peerTaskQueueEvent int

const (
	peerAdded   = peerTaskQueueEvent(1)
	peerRemoved = peerTaskQueueEvent(2)
)

type hookFunc func(p peer.ID, event peerTaskQueueEvent)

// PeerTaskQueue is a prioritized list of tasks to be executed on peers.
// Tasks are added to the queue, then popped off alternately between peers (roughly)
// to execute the block with the highest priority, or otherwise the one added
// first if priorities are equal.
type PeerTaskQueue struct {
	lock                      sync.Mutex
	peerComparator            peertracker.PeerComparator
	taskComparator            peertask.QueueTaskComparator
	pQueue                    pq.PQ
	peerTrackers              map[peer.ID]*peertracker.PeerTracker
	frozenPeers               map[peer.ID]struct{}
	hooks                     []hookFunc
	ignoreFreezing            bool
	taskMerger                peertracker.TaskMerger
	maxOutstandingWorkPerPeer int
}

// Option is a function that configures the peer task queue
type Option func(*PeerTaskQueue) Option

func chain(firstOption Option, secondOption Option) Option {
	return func(ptq *PeerTaskQueue) Option {
		firstReverse := firstOption(ptq)
		secondReverse := secondOption(ptq)
		return chain(secondReverse, firstReverse)
	}
}

// IgnoreFreezing is an option that can make the task queue ignore freezing and unfreezing
func IgnoreFreezing(ignoreFreezing bool) Option {
	return func(ptq *PeerTaskQueue) Option {
		previous := ptq.ignoreFreezing
		ptq.ignoreFreezing = ignoreFreezing
		return IgnoreFreezing(previous)
	}
}

// TaskMerger is an option that specifies merge behaviour when pushing a task
// with the same Topic as an existing Topic.
func TaskMerger(tmfp peertracker.TaskMerger) Option {
	return func(ptq *PeerTaskQueue) Option {
		previous := ptq.taskMerger
		ptq.taskMerger = tmfp
		return TaskMerger(previous)
	}
}

// MaxOutstandingWorkPerPeer is an option that specifies how many tasks a peer can have outstanding
// with the same Topic as an existing Topic.
func MaxOutstandingWorkPerPeer(count int) Option {
	return func(ptq *PeerTaskQueue) Option {
		previous := ptq.maxOutstandingWorkPerPeer
		ptq.maxOutstandingWorkPerPeer = count
		return MaxOutstandingWorkPerPeer(previous)
	}
}

func removeHook(hook hookFunc) Option {
	return func(ptq *PeerTaskQueue) Option {
		for i, testHook := range ptq.hooks {
			if &hook == &testHook {
				ptq.hooks = append(ptq.hooks[:i], ptq.hooks[i+1:]...)
				break
			}
		}
		return addHook(hook)
	}
}

func addHook(hook hookFunc) Option {
	return func(ptq *PeerTaskQueue) Option {
		ptq.hooks = append(ptq.hooks, hook)
		return removeHook(hook)
	}
}

// OnPeerAddedHook adds a hook function that gets called whenever the ptq adds a new peer
func OnPeerAddedHook(onPeerAddedHook func(p peer.ID)) Option {
	hook := func(p peer.ID, event peerTaskQueueEvent) {
		if event == peerAdded {
			onPeerAddedHook(p)
		}
	}
	return addHook(hook)
}

// OnPeerRemovedHook adds a hook function that gets called whenever the ptq adds a new peer
func OnPeerRemovedHook(onPeerRemovedHook func(p peer.ID)) Option {
	hook := func(p peer.ID, event peerTaskQueueEvent) {
		if event == peerRemoved {
			onPeerRemovedHook(p)
		}
	}
	return addHook(hook)
}

// PeerComparator is an option that specifies custom peer prioritization logic.
func PeerComparator(pc peertracker.PeerComparator) Option {
	return func(ptq *PeerTaskQueue) Option {
		previous := ptq.peerComparator
		ptq.peerComparator = pc
		return PeerComparator(previous)
	}
}

// TaskComparator is an option that specifies custom task prioritization logic.
func TaskComparator(tc peertask.QueueTaskComparator) Option {
	return func(ptq *PeerTaskQueue) Option {
		previous := ptq.taskComparator
		ptq.taskComparator = tc
		return TaskComparator(previous)
	}
}

// New creates a new PeerTaskQueue
func New(options ...Option) *PeerTaskQueue {
	ptq := &PeerTaskQueue{
		peerComparator: peertracker.DefaultPeerComparator,
		peerTrackers:   make(map[peer.ID]*peertracker.PeerTracker),
		frozenPeers:    make(map[peer.ID]struct{}),
		taskMerger:     &peertracker.DefaultTaskMerger{},
	}
	ptq.Options(options...)
	ptq.pQueue = pq.New(
		func(a, b pq.Elem) bool {
			pa := a.(*peertracker.PeerTracker)
			pb := b.(*peertracker.PeerTracker)
			return ptq.peerComparator(pa, pb)
		},
	)
	return ptq
}

// Options uses configuration functions to configure the peer task queue.
// It returns an Option that can be called to reverse the changes.
func (ptq *PeerTaskQueue) Options(options ...Option) Option {
	if len(options) == 0 {
		return nil
	}
	if len(options) == 1 {
		return options[0](ptq)
	}
	reverse := options[0](ptq)
	return chain(ptq.Options(options[1:]...), reverse)
}

func (ptq *PeerTaskQueue) callHooks(to peer.ID, event peerTaskQueueEvent) {
	for _, hook := range ptq.hooks {
		hook(to, event)
	}
}

// PeerTaskQueueStats captures current stats about the task queue.
type PeerTaskQueueStats struct {
	NumPeers   int
	NumActive  int
	NumPending int
}

// Stats returns current stats about the task queue.
func (ptq *PeerTaskQueue) Stats() *PeerTaskQueueStats {
	ptq.lock.Lock()
	defer ptq.lock.Unlock()

	s := &PeerTaskQueueStats{NumPeers: len(ptq.peerTrackers)}
	for _, t := range ptq.peerTrackers {
		ts := t.Stats()
		s.NumActive += ts.NumActive
		s.NumPending += ts.NumPending
	}
	return s
}

// PushTasks adds a new group of tasks for the given peer to the queue
func (ptq *PeerTaskQueue) PushTasks(to peer.ID, tasks ...peertask.Task) {
	ptq.lock.Lock()
	defer ptq.lock.Unlock()

	peerTracker, ok := ptq.peerTrackers[to]
	if !ok {
		var opts []peertracker.Option
		if ptq.taskComparator != nil {
			opts = append(opts, peertracker.WithQueueTaskComparator(ptq.taskComparator))
		}
		peerTracker = peertracker.New(to, ptq.taskMerger, ptq.maxOutstandingWorkPerPeer, opts...)
		ptq.pQueue.Push(peerTracker)
		ptq.peerTrackers[to] = peerTracker
		ptq.callHooks(to, peerAdded)
	}

	peerTracker.PushTasks(tasks...)
	ptq.pQueue.Update(peerTracker.Index())
}

// PopTasks finds the peer with the highest priority and pops as many tasks
// off the peer's queue as necessary to cover targetMinWork, in priority order.
// If there are not enough tasks to cover targetMinWork it just returns
// whatever is in the peer's queue.
// - Peers with the most "active" work are deprioritized.
//   This heuristic is for fairness, we try to keep all peers "busy".
// - Peers with the most "pending" work are prioritized.
//   This heuristic is so that peers with a lot to do get asked for work first.
// The third response argument is pending work: the amount of work in the
// queue for this peer.
func (ptq *PeerTaskQueue) PopTasks(targetMinWork int) (peer.ID, []*peertask.Task, int) {
	ptq.lock.Lock()
	defer ptq.lock.Unlock()

	if ptq.pQueue.Len() == 0 {
		return "", nil, -1
	}

	// Choose the highest priority peer
	peerTracker := ptq.pQueue.Peek().(*peertracker.PeerTracker)
	if peerTracker == nil {
		return "", nil, -1
	}

	// Get the highest priority tasks for the given peer
	out, pendingWork := peerTracker.PopTasks(targetMinWork)

	// If the peer has no more tasks, remove its peer tracker
	if peerTracker.IsIdle() {
		ptq.pQueue.Pop()
		target := peerTracker.Target()
		delete(ptq.peerTrackers, target)
		delete(ptq.frozenPeers, target)
		ptq.callHooks(target, peerRemoved)
	} else {
		// We may have modified the peer tracker's state (by popping tasks), so
		// update its position in the priority queue
		ptq.pQueue.Update(peerTracker.Index())
	}

	return peerTracker.Target(), out, pendingWork
}

// TasksDone is called to indicate that the given tasks have completed
// for the given peer
func (ptq *PeerTaskQueue) TasksDone(to peer.ID, tasks ...*peertask.Task) {
	ptq.lock.Lock()
	defer ptq.lock.Unlock()

	// Get the peer tracker for the peer
	peerTracker, ok := ptq.peerTrackers[to]
	if !ok {
		return
	}

	// Tell the peer tracker that the tasks have completed
	for _, task := range tasks {
		peerTracker.TaskDone(task)
	}

	// This may affect the peer's position in the peer queue, so update if
	// necessary
	ptq.pQueue.Update(peerTracker.Index())
}

// Remove removes a task from the queue.
func (ptq *PeerTaskQueue) Remove(topic peertask.Topic, p peer.ID) {
	ptq.lock.Lock()
	defer ptq.lock.Unlock()

	peerTracker, ok := ptq.peerTrackers[p]
	if ok {
		if peerTracker.Remove(topic) {
			// we now also 'freeze' that partner. If they sent us a cancel for a
			// block we were about to send them, we should wait a short period of time
			// to make sure we receive any other in-flight cancels before sending
			// them a block they already potentially have
			if !ptq.ignoreFreezing {
				if !peerTracker.IsFrozen() {
					ptq.frozenPeers[p] = struct{}{}
				}

				peerTracker.Freeze()
			}
			ptq.pQueue.Update(peerTracker.Index())
		}
	}
}

// FullThaw completely thaws all peers in the queue so they can execute tasks.
func (ptq *PeerTaskQueue) FullThaw() {
	ptq.lock.Lock()
	defer ptq.lock.Unlock()

	for p := range ptq.frozenPeers {
		peerTracker, ok := ptq.peerTrackers[p]
		if ok {
			peerTracker.FullThaw()
			delete(ptq.frozenPeers, p)
			ptq.pQueue.Update(peerTracker.Index())
		}
	}
}

// ThawRound unthaws peers incrementally, so that those have been frozen the least
// become unfrozen and able to execute tasks first.
func (ptq *PeerTaskQueue) ThawRound() {
	ptq.lock.Lock()
	defer ptq.lock.Unlock()

	for p := range ptq.frozenPeers {
		peerTracker, ok := ptq.peerTrackers[p]
		if ok {
			if peerTracker.Thaw() {
				delete(ptq.frozenPeers, p)
			}
			ptq.pQueue.Update(peerTracker.Index())
		}
	}
}
