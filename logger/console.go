package logger

import (
	"encoding/json"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/alibaba/sealer/common"
)

type brush func(string) string

func newBrush(color string) brush {
	pre := "\033["
	reset := "\033[0m"
	return func(text string) string {
		return pre + color + "m" + text + reset
	}
}

//鉴于终端的通常使用习惯，一般白色和黑色字体是不可行的,所以30,37不可用，
var colors = []brush{
	newBrush("1;41"), // Emergency          红色底
	newBrush("1;35"), // Alert              紫色
	newBrush("1;34"), // Critical           蓝色
	newBrush("1;31"), // Error              红色
	newBrush("1;33"), // Warn               黄色
	newBrush("1;36"), // Informational      天蓝色
	newBrush("1;32"), // Debug              绿色
	newBrush("1;32"), // Trace              绿色
}

type consoleLogger struct {
	stdOutMux sync.Mutex
	stdErrMux sync.Mutex
	Level     string `json:"level"`
	Colorful  bool   `json:"color"`
	LogLevel  logLevel
}

func (c *consoleLogger) Init(jsonConfig string) error {
	if len(jsonConfig) == 0 {
		return nil
	}

	err := json.Unmarshal([]byte(jsonConfig), c)
	if runtime.GOOS == common.WINDOWS {
		c.Colorful = false
	}

	if l, ok := LevelMap[c.Level]; ok {
		c.LogLevel = l
		return nil
	}

	return err
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
		c.printlnToStdErr(when, msg)
	default:
		c.printlnToStdOut(when, msg)
	}

	return nil
}

func (c *consoleLogger) Destroy() {

}

func (c *consoleLogger) printlnToStdOut(when time.Time, msg string) {
	c.stdOutMux.Lock()
	defer c.stdOutMux.Unlock()
	os.Stdout.Write(append([]byte(msg), '\n'))
}

func (c *consoleLogger) printlnToStdErr(when time.Time, msg string) {
	c.stdErrMux.Lock()
	defer c.stdErrMux.Unlock()
	os.Stderr.Write(append([]byte(msg), '\n'))
}

func init() {
	Register(AdapterConsole, &consoleLogger{
		LogLevel: LevelDebug,
		Colorful: runtime.GOOS != common.WINDOWS,
	})
}
