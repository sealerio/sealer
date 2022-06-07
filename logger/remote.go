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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type remoteLogger struct {
	sync.RWMutex

	TaskName string   `json:"taskName"`
	URL      string   `json:"url"`
	LogLevel logLevel `json:"logLevel"`

	URLPath string
}

func (f *remoteLogger) Init(jsonConfig string) error {
	if len(jsonConfig) == 0 {
		return nil
	}
	err := json.Unmarshal([]byte(jsonConfig), f)
	if err != nil {
		return err
	}

	reqURL, err := url.Parse(f.URL)
	if err != nil {
		return err
	}
	f.URLPath = reqURL.String()
	fmt.Println(f.URLPath)

	return err
}

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
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code from server: [%d] %s ", resp.StatusCode, resp.Status)
	}

	return nil
}

// LogWrite write logger message into file.
func (f *remoteLogger) LogWrite(when time.Time, msgText interface{}, level logLevel) error {
	msg, ok := msgText.(*loginfo)
	if !ok {
		return nil
	}

	if level > f.LogLevel {
		return nil
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name: uuid.New().String(),
		},
		InvolvedObject: corev1.ObjectReference{
			Name: f.TaskName,
		},
		Message: msg.Content,
		Type:    strconv.Itoa(int(level)),
	}

	bytesData, _ := json.Marshal(event)

	f.Lock()
	defer f.Unlock()

	fmt.Println(f.URLPath)
	fmt.Println(bytesData)

	return httpSend(f.URLPath, http.MethodPost, bytesData)
}

func (f *remoteLogger) Destroy() {}

func init() {
	Register(AdapterRemote, &remoteLogger{
		LogLevel: LevelInformational,
	})
}
