// Copyright © 2022 Alibaba Group Holding Ltd.
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

package logger

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestLogger_Print(t *testing.T) {
	if err := Init(LogOptions{
		LogToFile:    false,
		Verbose:      true,
		DisableColor: false,
	}); err != nil {
		panic(fmt.Sprintf("failed to init logger: %v\n", err))
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		logrus.Info("start to test log")
		for j := 0; j < 5; j++ {
			wg.Add(1)
			go func(x int) {
				time.Sleep(1 * time.Second)
				logrus.Debugf("i am the true entry %d", x)
				wg.Done()
			}(j)
		}
		wg.Wait()
	}
}
