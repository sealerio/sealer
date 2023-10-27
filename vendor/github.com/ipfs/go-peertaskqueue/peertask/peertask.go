package peertask

import (
	"time"

	pq "github.com/ipfs/go-ipfs-pq"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

type QueueTaskComparator func(a, b *QueueTask) bool

// FIFOCompare is a basic task comparator that returns tasks in the order created.
var FIFOCompare = func(a, b *QueueTask) bool {
	return a.created.Before(b.created)
}

// PriorityCompare respects the target peer's task priority. For tasks involving
// different peers, the oldest task is prioritized.
var PriorityCompare = func(a, b *QueueTask) bool {
	if a.Target == b.Target && a.Priority != b.Priority {
		return a.Priority > b.Priority
	}
	return FIFOCompare(a, b)
}

// WrapCompare wraps a QueueTask comparison function so it can be used as
// comparison for a priority queue
func WrapCompare(f QueueTaskComparator) func(a, b pq.Elem) bool {
	return func(a, b pq.Elem) bool {
		return f(a.(*QueueTask), b.(*QueueTask))
	}
}

// Topic is a non-unique name for a task. It's used by the client library
// to act on a task once it exits the queue.
type Topic interface{}

// Data is used by the client to associate extra information with a Task
type Data interface{}

// Task is a single task to be executed in Priority order.
type Task struct {
	// Topic for the task
	Topic Topic
	// Priority of the task
	Priority int
	// The size of the task
	// - peers with most active work are deprioritized
	// - peers with most pending work are prioritized
	Work int
	// Arbitrary data associated with this Task by the client
	Data Data
}

// QueueTask contains a Task, and also some bookkeeping information.
// It is used internally by the PeerTracker to keep track of tasks.
type QueueTask struct {
	Task
	Target  peer.ID
	created time.Time // created marks the time that the task was added to the queue
	index   int       // book-keeping field used by the pq container
}

// NewQueueTask creates a new QueueTask from the given Task.
func NewQueueTask(task Task, target peer.ID, created time.Time) *QueueTask {
	return &QueueTask{
		Task:    task,
		Target:  target,
		created: created,
	}
}

// Index implements pq.Elem.
func (pt *QueueTask) Index() int {
	return pt.index
}

// SetIndex implements pq.Elem.
func (pt *QueueTask) SetIndex(i int) {
	pt.index = i
}
