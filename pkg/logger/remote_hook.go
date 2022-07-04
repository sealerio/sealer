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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RemoteLogHook to send logs via remote URL.
type RemoteLogHook struct {
	sync.RWMutex

	TaskName string
	URL      string
}

func NewRemoteLogHook(remoteURL, taskName string) (*RemoteLogHook, error) {
	reqURL, err := url.Parse(remoteURL)
	if err != nil {
		return nil, err
	}

	return &RemoteLogHook{
		TaskName: taskName,
		URL:      reqURL.String(),
	}, err
}

// #nosec
func httpSend(url string, method string, body []byte) error {
	var resp *http.Response

	var err error
	switch method {
	case http.MethodGet:
		resp, err = http.Get(url)
	case http.MethodPost:
		resp, err = http.Post(url, "application/json", bytes.NewBuffer(body))
	}

	if err != nil {
		return fmt.Errorf("bad %s request to server : %w", method, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code from server: [%d] %s ", resp.StatusCode, resp.Status)
	}

	return nil
}

func (hook *RemoteLogHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}

	t := "Info"
	if entry.Level <= logrus.ErrorLevel {
		t = "Error"
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name: uuid.New().String(),
		},
		InvolvedObject: corev1.ObjectReference{
			Name: hook.TaskName,
		},
		Message: line,
		Type:    t,
	}

	bytesData, _ := json.Marshal(event)

	hook.Lock()
	defer hook.Unlock()

	return httpSend(hook.URL, http.MethodPost, bytesData)
}

func (hook *RemoteLogHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
}
