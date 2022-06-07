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
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sealerio/sealer/common"
)

type connLogger struct {
	sync.Mutex
	innerWriter    io.WriteCloser
	ReconnectOnMsg bool     `json:"reconnectOnMsg"`
	Reconnect      bool     `json:"reconnect"`
	Net            string   `json:"net"`
	Addr           string   `json:"addr"`
	LogLevel       logLevel `json:"logLevel"`
	illNetFlag     bool     //network exception flag
}

func (c *connLogger) Init(jsonConfig string) error {
	if len(jsonConfig) == 0 {
		return nil
	}
	fmt.Printf("consoleWriter Init:%s\n", jsonConfig)
	err := json.Unmarshal([]byte(jsonConfig), c)
	if err != nil {
		return err
	}

	if c.innerWriter != nil {
		err := c.innerWriter.Close()
		if err != nil {
			return err
		}
		c.innerWriter = nil
	}
	return nil
}

func (c *connLogger) LogWrite(when time.Time, msgText interface{}, level logLevel) (err error) {
	if level > c.LogLevel {
		return nil
	}

	msg, ok := msgText.(*loginfo)
	if !ok {
		return
	}

	if c.needToConnectOnMsg() {
		err = c.connect()
		if err != nil {
			return
		}
		//重连成功
		c.illNetFlag = false
	}

	//Each message is reconnected to the log center, which is suitable for service calls when the log writing frequency is extremely low, avoiding long-term connections and occupying resources
	if c.ReconnectOnMsg { // Do not enable frequent log sending
		defer c.innerWriter.Close()
	}

	//When the network is abnormal, the message is issued
	if !c.illNetFlag {
		err = c.println(msg)
		//Network exception, notify the go process that handles the network to automatically reconnect
		if err != nil {
			c.illNetFlag = true
		}
	}

	return
}

func (c *connLogger) Destroy() {
	if c.innerWriter != nil {
		err := c.innerWriter.Close()
		if err != nil {
			return
		}
	}
}

func (c *connLogger) connect() error {
	if c.innerWriter != nil {
		err := c.innerWriter.Close()
		if err != nil {
			return err
		}
		c.innerWriter = nil
	}
	addrs := strings.Split(c.Addr, ";")
	for _, addr := range addrs {
		conn, err := net.Dial(c.Net, addr)
		if err != nil {
			fmt.Fprintf(common.StdErr, "net.Dial error:%v\n", err)
			continue
			//return err
		}

		if tcpConn, ok := conn.(*net.TCPConn); ok {
			err = tcpConn.SetKeepAlive(true)
			if err != nil {
				fmt.Fprintf(common.StdErr, "failed to set tcp keep alive :%v\n", err)
				continue
			}
		}
		c.innerWriter = conn
		return nil
	}
	return fmt.Errorf("hava no valid logs service addr:%v", c.Addr)
}

func (c *connLogger) needToConnectOnMsg() bool {
	if c.Reconnect {
		c.Reconnect = false
		return true
	}

	if c.innerWriter == nil {
		return true
	}

	if c.illNetFlag {
		return true
	}
	return c.ReconnectOnMsg
}

func (c *connLogger) println(msg *loginfo) error {
	c.Lock()
	defer c.Unlock()
	ss, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = c.innerWriter.Write(append(ss, '\n'))

	//Return err to resolve automatic reconnection after log system network exception
	return err
}

func init() {
	Register(AdapterConn, &connLogger{LogLevel: LevelTrace})
}
