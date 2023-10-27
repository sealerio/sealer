package peertracker

import (
	"sync"

	"github.com/benbjohnson/clock"
	pq "github.com/ipfs/go-ipfs-pq"
	"github.com/ipfs/go-peertaskqueue/peertask"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

var clockInstance = clock.New()

// TaskMerger is an interface that is used to merge new tasks into the active
// and pending queues
type TaskMerger interface {
	// HasNewInfo indicates whether the given task has more information than
	// the existing group of tasks (which have the same Topic), and thus should
	// be merged.
	HasNewInfo(task peertask.Task, existing []*peertask.Task) bool
	// Merge copies relevant fields from a new task to an existing task.
	Merge(task peertask.Task, existing *peertask.Task)
}

// DefaultTaskMerger is the TaskMerger used by default. It never overwrites an
// existing task (with the same Topic).
type DefaultTaskMerger struct{}

func (*DefaultTaskMerger) HasNewInfo(task peertask.Task, existing []*peertask.Task) bool {
	return false
}

func (*DefaultTaskMerger) Merge(task peertask.Task, existing *peertask.Task) {
}

// PeerTracker tracks task blocks for a single peer, as well as active tasks
// for that peer
type PeerTracker struct {
	target peer.ID

	// Tasks that are pending being made active
	pendingTasks map[peertask.Topic]*peertask.QueueTask

	activelk sync.Mutex
	// Tasks that have been made active. Unfortuantely, we can have multiple for the same topic
	// as we might get a "supperior" request after starting to handle the initial one.
	activeTasks map[peertask.Topic][]*peertask.Task
	activeWork  int

	maxActiveWorkPerPeer int

	// for the PQ interface
	index int

	freezeVal int

	queueTaskComparator peertask.QueueTaskComparator

	// priority queue of tasks belonging to this peer
	taskQueue pq.PQ

	taskMerger TaskMerger
}

// Option is a function that configures the peer tracker
type Option func(*PeerTracker)

// WithQueueTaskComparator sets a custom QueueTask comparison function for the
// peer tracker's task queue.
func WithQueueTaskComparator(f peertask.QueueTaskComparator) Option {
	return func(pt *PeerTracker) {
		pt.queueTaskComparator = f
	}
}

// New creates a new PeerTracker
func New(target peer.ID, taskMerger TaskMerger, maxActiveWorkPerPeer int, opts ...Option) *PeerTracker {
	pt := &PeerTracker{
		target:               target,
		queueTaskComparator:  peertask.PriorityCompare,
		pendingTasks:         make(map[peertask.Topic]*peertask.QueueTask),
		activeTasks:          make(map[peertask.Topic][]*peertask.Task),
		taskMerger:           taskMerger,
		maxActiveWorkPerPeer: maxActiveWorkPerPeer,
	}

	for _, opt := range opts {
		opt(pt)
	}

	pt.taskQueue = pq.New(peertask.WrapCompare(pt.queueTaskComparator))

	return pt
}

// PeerComparator is used for peer prioritization.
// It should return true if peer 'a' has higher priority than peer 'b'
type PeerComparator func(a, b *PeerTracker) bool

// DefaultPeerComparator implements the default peer prioritization logic.
func DefaultPeerComparator(pa, pb *PeerTracker) bool {
	// having no pending tasks means lowest priority
	paPending := len(pa.pendingTasks)
	pbPending := len(pb.pendingTasks)
	if paPending == 0 {
		return false
	}
	if pbPending == 0 {
		return true
	}

	// Frozen peers have lowest priority
	if pa.freezeVal > pb.freezeVal {
		return false
	}
	if pa.freezeVal < pb.freezeVal {
		return true
	}

	// If each peer has an equal amount of work in its active queue, choose the
	// peer with the most amount of work pending
	if pa.activeWork == pb.activeWork {
		return paPending > pbPending
	}

	// Choose the peer with the least amount of work in its active queue.
	// This way we "keep peers busy" by sending them as much data as they can
	// process.
	return pa.activeWork < pb.activeWork
}

// TaskPriorityPeerComparator prioritizes peers based on their highest priority task.
func TaskPriorityPeerComparator(comparator peertask.QueueTaskComparator) PeerComparator {
	return func(pa, pb *PeerTracker) bool {
		ta := pa.taskQueue.Peek()
		tb := pb.taskQueue.Peek()
		if ta == nil {
			return false
		}
		if tb == nil {
			return true
		}

		return comparator(ta.(*peertask.QueueTask), tb.(*peertask.QueueTask))
	}
}

// Target returns the peer that this peer tracker tracks tasks for
func (p *PeerTracker) Target() peer.ID {
	return p.target
}

// IsIdle returns true if the peer has no active tasks or queued tasks
func (p *PeerTracker) IsIdle() bool {
	p.activelk.Lock()
	defer p.activelk.Unlock()

	return len(p.pendingTasks) == 0 && len(p.activeTasks) == 0
}

// PeerTrackerStats captures number of active and pending tasks for this peer.
type PeerTrackerStats struct {
	NumPending int
	NumActive  int
}

// Stats returns current statistics for this peer.
func (p *PeerTracker) Stats() *PeerTrackerStats {
	p.activelk.Lock()
	defer p.activelk.Unlock()
	return &PeerTrackerStats{NumPending: len(p.pendingTasks), NumActive: len(p.activeTasks)}
}

// Index implements pq.Elem.
func (p *PeerTracker) Index() int {
	return p.index
}

// SetIndex implements pq.Elem.
func (p *PeerTracker) SetIndex(i int) {
	p.index = i
}

// PushTasks adds a group of tasks onto a peer's queue
func (p *PeerTracker) PushTasks(tasks ...peertask.Task) {
	now := clockInstance.Now()

	p.activelk.Lock()
	defer p.activelk.Unlock()

	for _, task := range tasks {
		// If the new task doesn't add any more information over what we
		// already have in the active queue, then we can skip the new task
		if !p.taskHasMoreInfoThanActiveTasks(task) {
			continue
		}

		// If there is already a non-active task with this Topic
		if existingTask, ok := p.pendingTasks[task.Topic]; ok {
			// If the new task has a higher priority than the old task,
			if task.Priority > existingTask.Priority {
				// Update the priority and the task's position in the queue
				existingTask.Priority = task.Priority
				p.taskQueue.Update(existingTask.Index())
			}

			p.taskMerger.Merge(task, &existingTask.Task)

			// A task with the Topic exists, so we don't need to add
			// the new task to the queue
			continue
		}

		// Push the new task onto the queue
		qTask := peertask.NewQueueTask(task, p.target, now)
		p.pendingTasks[task.Topic] = qTask
		p.taskQueue.Push(qTask)
	}
}

// PopTasks pops as many tasks off the queue as necessary to cover
// targetMinWork, in priority order. If there are not enough tasks to cover
// targetMinWork it just returns whatever is in the queue.
// The second response argument is pending work: the amount of work in the
// queue for this peer.
func (p *PeerTracker) PopTasks(targetMinWork int) ([]*peertask.Task, int) {
	var out []*peertask.Task
	work := 0
	for p.taskQueue.Len() > 0 && p.freezeVal == 0 && work < targetMinWork {
		if p.maxActiveWorkPerPeer > 0 {
			// Do not add work to a peer that is already maxed out
			p.activelk.Lock()
			activeWork := p.activeWork
			p.activelk.Unlock()
			if activeWork >= p.maxActiveWorkPerPeer {
				break
			}
		}

		// Pop the next task off the queue
		t := p.taskQueue.Pop().(*peertask.QueueTask)

		// Start the task (this makes it "active")
		p.startTask(&t.Task)

		out = append(out, &t.Task)
		work += t.Work
	}

	return out, p.getPendingWork()
}

// startTask signals that a task was started for this peer.
func (p *PeerTracker) startTask(task *peertask.Task) {
	p.activelk.Lock()
	defer p.activelk.Unlock()

	// Remove task from pending queue
	delete(p.pendingTasks, task.Topic)

	// Add task to active queue
	if _, ok := p.activeTasks[task]; !ok {
		p.activeTasks[task.Topic] = append(p.activeTasks[task.Topic], task)
		p.activeWork += task.Work
	}
}

func (p *PeerTracker) getPendingWork() int {
	total := 0
	for _, t := range p.pendingTasks {
		total += t.Work
	}
	return total
}

// TaskDone signals that a task was completed for this peer.
func (p *PeerTracker) TaskDone(task *peertask.Task) {
	p.activelk.Lock()
	defer p.activelk.Unlock()

	// Remove task from active queue
	activeTasks, ok := p.activeTasks[task.Topic]
	if !ok {
		return
	}
	// There will usually be 0 through 2 of these, so this should always be fast.
	newTasks := activeTasks[:0]
	for _, t := range activeTasks {
		if task == t {
			p.activeWork -= t.Work
			continue
		}
		newTasks = append(newTasks, t)
	}

	if p.activeWork < 0 {
		panic("more tasks finished than started!")
	}

	if len(newTasks) == 0 {
		delete(p.activeTasks, task.Topic)
	} else {
		// Garbage collection.
		for i := len(newTasks); i < len(activeTasks); i++ {
			activeTasks[i] = nil
		}

		p.activeTasks[task.Topic] = newTasks
	}
}

// Remove removes the task with the given topic from this peer's queue
func (p *PeerTracker) Remove(topic peertask.Topic) bool {
	t, ok := p.pendingTasks[topic]
	if ok {
		delete(p.pendingTasks, topic)
		p.taskQueue.Remove(t.Index())
	}
	return ok
}

// Freeze increments the freeze value for this peer. While a peer is frozen
// (freeze value > 0) it will not execute tasks.
func (p *PeerTracker) Freeze() {
	p.freezeVal++
}

// Thaw decrements the freeze value for this peer. While a peer is frozen
// (freeze value > 0) it will not execute tasks.
func (p *PeerTracker) Thaw() bool {
	p.freezeVal -= (p.freezeVal + 1) / 2
	return p.freezeVal <= 0
}

// FullThaw completely unfreezes this peer so it can execute tasks.
func (p *PeerTracker) FullThaw() {
	p.freezeVal = 0
}

// IsFrozen returns whether this peer is frozen and unable to execute tasks.
func (p *PeerTracker) IsFrozen() bool {
	return p.freezeVal > 0
}

// Indicates whether the new task adds any more information over tasks that are
// already in the active task queue
func (p *PeerTracker) taskHasMoreInfoThanActiveTasks(task peertask.Task) bool {
	tasksWithTopic := p.activeTasks[task.Topic]

	// No tasks with that topic, so the new task adds information
	if len(tasksWithTopic) == 0 {
		return true
	}

	return p.taskMerger.HasNewInfo(task, tasksWithTopic)
}
