package taskqueue

import (
	"context"
	"time"

	"github.com/ipfs/go-peertaskqueue"
	"github.com/ipfs/go-peertaskqueue/peertask"
	peer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
)

const thawSpeed = time.Millisecond * 100

// Executor runs a single task on the queue
type Executor interface {
	ExecuteTask(ctx context.Context, pid peer.ID, task *peertask.Task) bool
}

type TaskQueue interface {
	PushTask(p peer.ID, task peertask.Task)
	TaskDone(p peer.ID, task *peertask.Task)
	Remove(t peertask.Topic, p peer.ID)
	Stats() graphsync.RequestStats
}

// TaskQueue is a wrapper around peertaskqueue.PeerTaskQueue that manages running workers
// that pop tasks and execute them
type WorkerTaskQueue struct {
	ctx           context.Context
	cancelFn      func()
	peerTaskQueue *peertaskqueue.PeerTaskQueue
	workSignal    chan struct{}
	ticker        *time.Ticker
}

// NewTaskQueue initializes a new queue
func NewTaskQueue(ctx context.Context) *WorkerTaskQueue {
	ctx, cancelFn := context.WithCancel(ctx)
	return &WorkerTaskQueue{
		ctx:           ctx,
		cancelFn:      cancelFn,
		peerTaskQueue: peertaskqueue.New(),
		workSignal:    make(chan struct{}, 1),
		ticker:        time.NewTicker(thawSpeed),
	}
}

// PushTask pushes a new task on to the queue
func (tq *WorkerTaskQueue) PushTask(p peer.ID, task peertask.Task) {
	tq.peerTaskQueue.PushTasks(p, task)
	select {
	case tq.workSignal <- struct{}{}:
	default:
	}
}

// TaskDone marks a task as completed so further tasks can be executed
func (tq *WorkerTaskQueue) TaskDone(p peer.ID, task *peertask.Task) {
	tq.peerTaskQueue.TasksDone(p, task)
}

// Stats returns statistics about a task queue
func (tq *WorkerTaskQueue) Stats() graphsync.RequestStats {
	ptqstats := tq.peerTaskQueue.Stats()
	return graphsync.RequestStats{
		TotalPeers: uint64(ptqstats.NumPeers),
		Active:     uint64(ptqstats.NumActive),
		Pending:    uint64(ptqstats.NumPending),
	}
}

// Remove removes a task from the execution queue
func (tq *WorkerTaskQueue) Remove(topic peertask.Topic, p peer.ID) {
	tq.peerTaskQueue.Remove(topic, p)
}

// Startup runs the given number of task workers with the given executor
func (tq *WorkerTaskQueue) Startup(workerCount uint64, executor Executor) {
	for i := uint64(0); i < workerCount; i++ {
		go tq.worker(executor)
	}
}

// Shutdown shuts down all running workers
func (tq *WorkerTaskQueue) Shutdown() {
	tq.cancelFn()
}

func (tq *WorkerTaskQueue) worker(executor Executor) {
	targetWork := 1
	for {
		pid, tasks, _ := tq.peerTaskQueue.PopTasks(targetWork)
		for len(tasks) == 0 {
			select {
			case <-tq.ctx.Done():
				return
			case <-tq.workSignal:
				pid, tasks, _ = tq.peerTaskQueue.PopTasks(targetWork)
			case <-tq.ticker.C:
				tq.peerTaskQueue.ThawRound()
				pid, tasks, _ = tq.peerTaskQueue.PopTasks(targetWork)
			}
		}
		for _, task := range tasks {
			terminate := executor.ExecuteTask(tq.ctx, pid, task)
			if terminate {
				return
			}
		}
	}
}
