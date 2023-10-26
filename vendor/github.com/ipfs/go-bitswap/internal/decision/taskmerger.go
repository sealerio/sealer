package decision

import (
	"github.com/ipfs/go-peertaskqueue/peertask"
)

// taskData is extra data associated with each task in the request queue
type taskData struct {
	// Tasks can be want-have or want-block
	IsWantBlock bool
	// Whether to immediately send a response if the block is not found
	SendDontHave bool
	// The size of the block corresponding to the task
	BlockSize int
	// Whether the block was found
	HaveBlock bool
}

type taskMerger struct{}

func newTaskMerger() *taskMerger {
	return &taskMerger{}
}

// The request queue uses this Method to decide if a newly pushed task has any
// new information beyond the tasks with the same Topic (CID) in the queue.
func (*taskMerger) HasNewInfo(task peertask.Task, existing []*peertask.Task) bool {
	haveSize := false
	isWantBlock := false
	for _, et := range existing {
		etd := et.Data.(*taskData)
		if etd.HaveBlock {
			haveSize = true
		}

		if etd.IsWantBlock {
			isWantBlock = true
		}
	}

	// If there is no active want-block and the new task is a want-block,
	// the new task is better
	newTaskData := task.Data.(*taskData)
	if !isWantBlock && newTaskData.IsWantBlock {
		return true
	}

	// If there is no size information for the CID and the new task has
	// size information, the new task is better
	if !haveSize && newTaskData.HaveBlock {
		return true
	}

	return false
}

// The request queue uses Merge to merge a newly pushed task with an existing
// task with the same Topic (CID)
func (*taskMerger) Merge(task peertask.Task, existing *peertask.Task) {
	newTask := task.Data.(*taskData)
	existingTask := existing.Data.(*taskData)

	// If we now have block size information, update the task with
	// the new block size
	if !existingTask.HaveBlock && newTask.HaveBlock {
		existingTask.HaveBlock = newTask.HaveBlock
		existingTask.BlockSize = newTask.BlockSize
	}

	// If replacing a want-have with a want-block
	if !existingTask.IsWantBlock && newTask.IsWantBlock {
		// Change the type from want-have to want-block
		existingTask.IsWantBlock = true
		// If the want-have was a DONT_HAVE, or the want-block has a size
		if !existingTask.HaveBlock || newTask.HaveBlock {
			// Update the entry size
			existingTask.HaveBlock = newTask.HaveBlock
			existing.Work = task.Work
		}
	}

	// If the task is a want-block, make sure the entry size is equal
	// to the block size (because we will send the whole block)
	if existingTask.IsWantBlock && existingTask.HaveBlock {
		existing.Work = existingTask.BlockSize
	}
}
