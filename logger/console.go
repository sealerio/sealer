// Copyright © 2021 github.com/wonderivan/logger
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
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"github.com/sealerio/sealer/common"
)

type brush func(string) string

func newBrush(color string) brush {
	pre := "\033["
	reset := "\033[0m"
	return func(text string) string {
		return pre + color + "m" + text + reset
	}
}

//In view of the usual usage habits of the terminal, generally white and black fonts are not feasible, so 30,37 are not available,
var colors = []brush{
	newBrush("1;41"), // Emergency          red bottom
	newBrush("1;35"), // Alert              Purple
	newBrush("1;34"), // Critical           blue
	newBrush("1;31"), // Error              red
	newBrush("1;33"), // Warn               yellow
	newBrush("1;36"), // Informational      sky blue
	newBrush("1;32"), // Debug              green
	newBrush("1;32"), // Trace              green
}

type consoleLogger struct {
	stdOutMux sync.Mutex
	stdErrMux sync.Mutex
	Colorful  bool     `json:"color"`
	LogLevel  logLevel `json:"logLevel"`
}

func (c *consoleLogger) Init(jsonConfig string) error {
	if len(jsonConfig) == 0 {
		return nil
	}

	if err := json.Unmarshal([]byte(jsonConfig), c); err != nil {
		return err
	}

	if runtime.GOOS == common.WINDOWS {
		c.Colorful = false
	}

	return nil
}

func (c *consoleLogger) LogWrite(when time.Time, msgText interface{}, level logLevel) error {
	if level > c.LogLevel {
		return nil
	}
	msg, ok := msgText.(string)
	if !ok {
		return nil
	}
	if c.Colorful {
		msg = colors[level](msg)
	}
	switch level {
	case LevelEmergency, LevelAlert, LevelCritical, LevelError:
		c.printlnToStdErr(msg)
	default:
		c.printlnToStdOut(msg)
	}

	return nil
}

func (c *consoleLogger) Destroy() {

}

func (c *consoleLogger) printlnToStdOut(msg string) {
	c.stdOutMux.Lock()
	defer c.stdOutMux.Unlock()
	_, _ = common.StdOut.Write(append([]byte(msg), '\n'))
}

func (c *consoleLogger) printlnToStdErr(msg string) {
	c.stdErrMux.Lock()
	defer c.stdErrMux.Unlock()
	_, _ = common.StdErr.Write(append([]byte(msg), '\n'))
}

func init() {
	Register(AdapterConsole, &consoleLogger{
		LogLevel: LevelDebug,
		Colorful: runtime.GOOS != common.WINDOWS,
	})
}
