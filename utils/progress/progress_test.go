package progress

import (
	"testing"
	"time"
)

func TestChannelProgressBar(t *testing.T) {
	var total int64 = 256
	ch := make(chan Msg, 256)

	flow := NewProgressFlow()
	flow.AddProgressTasks(TaskDef{ProgressSrc: ChannelTask{
		ProgressChan: ch,
	}, Task: "Downloading", Job: "Job", Max: total, SuccessMsg: "success", FailMsg: "failed"})

	go func() {
		for i := 0; int64(i) < total; i++ {
			//if int64(i) == 20 {
			//	//close(ch)
			//	ch <- ProgressMsg{Status: StatusFail, Msg: "failed"}
			//	break
			//}

			ch <- Msg{Inc: 1, Status: StatusPlain}
			time.Sleep(50 * time.Millisecond)
		}
	}()
	flow.Start()
}
