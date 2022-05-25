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

const (
	WINDOWS = "windows"
)

const (
	DefaultLogDir        = "/var/lib/sealer/log"
	LogTimeDefaultFormat = "2006-01-02 15:04:05"
	AdapterConsole       = "console"
	AdapterFile          = "file"
	AdapterConn          = "conn"
)

// Log level, from 0-7, daily priority from high to low
const (
	LevelEmergency     logLevel = iota // System level emergency, such as disk error, memory exception, network unavailable, etc.
	LevelAlert                         // System-level warnings, such as database access exceptions, configuration file errors, etc.
	LevelCritical                      // System-level dangers, such as permission errors, access exceptions, etc.
	LevelError                         // User level error
	LevelWarning                       // User level warning
	LevelInformational                 // User level information
	LevelDebug                         // User level debugging
	LevelTrace                         // User level basic output
)
