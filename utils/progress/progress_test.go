// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
