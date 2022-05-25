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

package common

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sealerio/sealer/logger"
)

var (
	StdOut      = os.Stdout
	StdErr      = os.Stderr
	AuditStdOut = &AuditStd{os.Stdout}
	AuditStdErr = &AuditStd{os.Stderr}
)

type AuditStd struct {
	*os.File
}

func (a *AuditStd) Write(b []byte) (n int, err error) {
	n, err = fmt.Fprintln(a.File, string(b))
	if err != nil {
		return
	}
	if err := a.writeToLogFile(b); err != nil {
		fmt.Fprintf(StdErr, "failed to audit: %v\n", err)
	}
	return
}

func (a *AuditStd) WriteToLogFile(b []byte) {
	if err := a.writeToLogFile(b); err != nil {
		fmt.Fprintf(StdErr, "failed to write log file: %v\n", err)
	}
}

func (a *AuditStd) WriteString(b string) (n int, err error) {
	n, err = a.File.WriteString(b)
	if err != nil {
		return
	}
	if err := a.writeToLogFile([]byte(b)); err != nil {
		fmt.Fprintf(StdErr, "failed to write log file: %v\n", err)
	}
	return
}

func (a *AuditStd) writeToLogFile(b []byte) error {
	bf := bytes.NewBuffer(b)
	auditFile := logger.GetLoggerFileName()
	write, err := os.OpenFile(filepath.Clean(auditFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %v", filepath.Clean(auditFile), err)
	}
	defer func() {
		if err := write.Close(); err != nil {
			fmt.Fprintf(StdErr, "Error closing file: %s\n", err)
		}
	}()
	scanner := bufio.NewScanner(bf)
	for scanner.Scan() {
		if _, err := write.WriteString(fmt.Sprintf("%s\n", scanner.Text())); err != nil {
			return fmt.Errorf("failed to write string: %v", err)
		}
	}
	return nil
}
