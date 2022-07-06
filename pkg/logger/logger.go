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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type LogOptions struct {
	// sealer log file path, default log directory is `/var/lib/sealer/log`
	OutputPath string
	// Verbose: sealer log level,if it is ture will set debug log mode.
	Verbose bool
	// DisableColor if true will disable outputting colors.
	DisableColor         bool
	RemoteLoggerURL      string
	RemoteLoggerTaskName string
	// LogToFile flag represent whether write log to disk, default is false.
	LogToFile bool
}

func Init(options LogOptions) error {
	if options.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	logrus.SetReportCaller(true)

	logrus.SetFormatter(&Formatter{
		DisableColor: options.DisableColor,
	})

	if options.LogToFile {
		fh, err := NewFileHook(options.OutputPath)
		if err != nil {
			return errors.Errorf("failed to init log file hook: %v", err)
		}
		logrus.AddHook(fh)
	}

	if options.RemoteLoggerURL != "" {
		rl, err := NewRemoteLogHook(options.RemoteLoggerURL, options.RemoteLoggerTaskName)
		if err != nil {
			return errors.Errorf("failed to init log remote hook: %v", err)
		}
		logrus.AddHook(rl)
	}

	return nil
}
