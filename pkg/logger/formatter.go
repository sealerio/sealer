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
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	colorRed    = 31
	colorYellow = 33
	colorBlue   = 36
	colorGray   = 37
)

const (
	defaultTimestampFormat = "2006-01-02 15:04:05"
)

func getColorByLevel(level logrus.Level) int {
	switch level {
	case logrus.DebugLevel, logrus.TraceLevel:
		return colorGray
	case logrus.WarnLevel:
		return colorYellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return colorRed
	default:
		return colorBlue
	}
}

type Formatter struct {
	// DisableColor disable colors
	DisableColor bool
	// HideLogTime if send to remote log system that already adds timestamps.
	HideLogTime bool
	// HideLogPath more simple log message without file and lines
	HideLogPath     bool
	TimestampFormat string
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	if !f.HideLogTime {
		b.WriteString(entry.Time.Format(timestampFormat))
	}

	levelStr := strings.ToUpper(entry.Level.String())

	newLog := fmt.Sprintf(" [%s] %s\n", levelStr, entry.Message)

	if !f.HideLogPath {
		if entry.HasCaller() {
			fName := filepath.Base(entry.Caller.File)
			newLog = fmt.Sprintf(" [%s] [%s:%d] %s\n", levelStr, fName, entry.Caller.Line, entry.Message)
		}
	}

	if !f.DisableColor {
		levelColor := getColorByLevel(entry.Level)
		//here is the console color format specification example:
		//var Reset = "\033[0m"
		//var Red = "\033[31m"
		//var Green = "\033[32m"
		//var Yellow = "\033[33m"
		//var Blue = "\033[34m"
		//var Purple = "\033[35m"
		//var Cyan = "\033[36m"
		//var Gray = "\033[37m"
		//var White = "\033[97m"

		fmt.Fprintf(b, "\033[%dm%s\033[0m", levelColor, newLog)
	} else {
		b.WriteString(newLog)
	}

	b.WriteByte('\n')

	return b.Bytes(), nil
}
