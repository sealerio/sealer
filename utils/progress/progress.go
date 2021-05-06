package progress

import (
	"sync"

	"github.com/alibaba/sealer/utils"
	"github.com/pkg/errors"
	"github.com/vbauerster/mpb/v6"
	"github.com/vbauerster/mpb/v6/decor"
)

const (
	messageBufferMax = 1
	progressWidth    = 60
	int64Max         = 0x7FFFFFFFFFFFFFFF
)

const (
	StatusPlain = iota
	StatusSuccess
	StatusFail
)

type TaskDef struct {
	Task        string // left message, only used to show message
	Job         string // message right next to Task
	Max         int64  // process max value
	ProgressSrc Task   // real Task definition
	SuccessMsg  string
	FailMsg     string
	ID          string // id is majorly used to make all transactions with same id share a common bar
	CleanOnDone bool   // used to clean bar after done
}

type Msg struct {
	Inc    int64
	Status int8
	Msg    string
}

type Flow struct {
	mux        sync.Mutex
	progress   *mpb.Progress
	processDef map[string][]TaskDef // pre definition for all task
	allBars    map[string][]*mpb.Bar
}

func (flow *Flow) AddProgressTasks(tasks ...TaskDef) *Flow {
	flow.mux.Lock()
	defer flow.mux.Unlock()
	if len(tasks) == 0 {
		panic("tasks should be provided")
	}

	lastID := tasks[0].ID
	uid := lastID
	if uid == "" {
		uid = utils.GenUniqueID(8)
	}
	for _, task := range tasks {
		//TODO mux
		if lastID != task.ID {
			panic("failed to add progress task, err: appending tasks within a operation should have same id")
		}
		task.ID = uid
		flow.processDef[uid] = append(flow.processDef[uid], task)
		if task.SuccessMsg != "" {
			flow.processDef[uid] = append(flow.processDef[uid], TaskDef{ID: uid, Max: 1, SuccessMsg: task.SuccessMsg, ProgressSrc: successMsgTask{}})
		}
	}

	return flow
}

func (flow *Flow) registryProcessBar(def TaskDef, addProgressBar func(dec decor.Decorator, taskDef TaskDef) *mpb.Bar) (job processJob) {
	var bar *mpb.Bar
	switch (def.ProgressSrc).(type) {
	case ChannelTask:
		task := (def.ProgressSrc).(ChannelTask)
		bar = addProgressBar(decor.CountersNoUnit("%d/%d", decor.WCSyncWidth), def)
		job = processJob{
			function: func(cxt Context) error {
				curBar := cxt.GetCurrentBar()
				if curBar == nil {
					return errors.New("failed to execute job, err: current bar not found")
				}
				for msg := range task.ProgressChan {
					if msg.Status == StatusPlain {
						curBar.IncrInt64(msg.Inc)
						if curBar.Completed() {
							return nil
						}
						continue
					}
					if msg.Status == StatusFail {
						return errors.New(def.FailMsg + ":" + msg.Msg)
					}
				}
				return nil
			},
		}
	case TakeOverTask:
		task := def.ProgressSrc.(TakeOverTask)
		bar = addProgressBar(decor.CountersKibiByte("%.2f/%.2f"), def)
		job = processJob{
			function: func(cxt Context) error {
				return task.Action(cxt)
			},
		}
	case successMsgTask:
		bar = flow.addMessageBar(flow.tailBar(def.ID), def.SuccessMsg, "")
		job = processJob{
			function: func(cxt Context) error {
				curBar := cxt.GetCurrentBar()
				if curBar == nil {
					return errors.New("failed to get current message bar")
				}
				return nil
			},
		}
	default:
		panic("unsupported progress src data type")
	}

	flow.allBars[def.ID] = append(flow.allBars[def.ID], bar)
	job.cxt = def.ProgressSrc.context().WithCurrentProcessBar(bar)
	return
}

// add real progress bar
func (flow *Flow) constructFinalBar() map[string][]processJob {
	for _, defs := range flow.processDef {
		err := validateTaskDef(defs)
		if err != nil {
			panic(err)
		}
	}

	addProgressBar := func(dec decor.Decorator, taskDef TaskDef) *mpb.Bar {
		var removeOnComplete mpb.BarOption
		if taskDef.CleanOnDone {
			removeOnComplete = mpb.BarRemoveOnComplete()
		}
		return flow.progress.Add(taskDef.Max,
			nil,
			removeOnComplete,
			mpb.BarQueueAfter(flow.tailBar(taskDef.ID)),
			mpb.PrependDecorators(
				decor.Name(taskDef.Task, decor.WC{W: len(taskDef.Task) + 1, C: decor.DidentRight}),
				decor.Name(taskDef.Job, decor.WCSyncSpaceR),
				dec,
			))
	}

	var jobs = make(map[string][]processJob)
	for _, defs := range flow.processDef {
		for _, def := range defs {
			jobs[def.ID] = append(jobs[def.ID], flow.registryProcessBar(def, addProgressBar))
		}
	}
	return jobs
}

func (flow *Flow) startExecuteJobs(jobs map[string][]processJob) {
	for id, pjobs := range jobs {
		go func(id string, js []processJob) {
			cxt := Context{}
			for _, job := range js {
				cxt.CopyAllVar(job.cxt)
				err := job.function(cxt)
				if err != nil {
					flow.appendErrorMessageBar(flow.allBars[id], err.Error(), "")
					break
				}
				curBar := job.cxt.GetCurrentBar()
				if curBar != nil {
					curBar.SetCurrent(int64Max)
				}
			}
		}(id, pjobs)
	}
}

func validateTaskDef(defs []TaskDef) error {
	for _, def := range defs {
		if def.ProgressSrc == nil {
			return errors.New("falied to validate task def, err: progress src should be provided")
		}
		if def.Max <= 0 {
			return errors.New("failed to validate task def, err: def max is less or equal to 0")
		}
		switch def.ProgressSrc.(type) {
		case ChannelTask:
			if (def.ProgressSrc).(ChannelTask).ProgressChan == nil {
				return errors.New("progress chan should be provided")
			}
		case TakeOverTask:
			src := def.ProgressSrc.(TakeOverTask)
			if src.Cxt == nil || src.Action == nil {
				return errors.New("both cxt and action should be provided")
			}
		case successMsgTask:
		default:
			return errors.New("unsupported progress src")
		}
	}

	return nil
}

func (flow *Flow) ShowMessage(msg string, bar *mpb.Bar) *mpb.Bar {
	newBar := flow.addMessageBar(bar, msg, "")
	newBar.SetCurrent(int64Max)
	return newBar
}

func (flow *Flow) appendErrorMessageBar(preBars []*mpb.Bar, task, job string) {
	if preBars == nil {
		preBars = []*mpb.Bar{nil}
	}
	preBars = append(preBars, flow.addMessageBar(preBars[len(preBars)-1], task, job))
	for _, b := range preBars {
		if b != nil {
			// all the previous complete messages are fake news
			b.SetCurrent(int64Max)
		}
	}
}

func (flow *Flow) addMessageBar(bar *mpb.Bar, task, job string) *mpb.Bar {
	return flow.progress.Add(messageBufferMax,
		nil,
		mpb.BarQueueAfter(bar),
		mpb.PrependDecorators(
			decor.Name(task, decor.WC{W: len(task) + 1, C: decor.DidentRight}),
			decor.Name(job, decor.WCSyncSpaceR),
		))
}

func (flow *Flow) tailBar(barID string) *mpb.Bar {
	bars := flow.allBars[barID]
	if len(bars) > 0 {
		return bars[len(bars)-1]
	}
	return nil
}

func NewProgressFlow() *Flow {
	return &Flow{progress: mpb.New(mpb.WithWidth(progressWidth)),
		allBars:    make(map[string][]*mpb.Bar),
		processDef: make(map[string][]TaskDef)}
}

func (flow *Flow) Start() {
	jobs := flow.constructFinalBar()
	flow.startExecuteJobs(jobs)
	flow.progress.Wait()
}
