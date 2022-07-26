// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/2/22 10:26 下午
// @File : error-recorder
//

package utils

import (
	"fmt"
	"strings"
	"sync"
)

type HostErrRecorder struct {
	errMsgs map[string]string
	lock    sync.Mutex
}

func NewHostErrRecorder() *HostErrRecorder {
	return &HostErrRecorder{
		errMsgs: make(map[string]string),
	}
}

func (r *HostErrRecorder) Append(host, errMsg string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.errMsgs[host] = errMsg
}

func (r *HostErrRecorder) AppendErr(host string, err error) {
	if err == nil {
		return
	}
	r.Append(host, err.Error())
}

func (r *HostErrRecorder) Result() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if len(r.errMsgs) == 0 {
		return nil
	}

	var errMsgSlice []string
	for k, v := range r.errMsgs {
		errMsgSlice = append(errMsgSlice, fmt.Sprintf("host %s: %s", k, v))
	}

	return fmt.Errorf("%s", strings.Join(errMsgSlice, "\n"))
}

func (r *HostErrRecorder) ErrMsgs() map[string]string {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.errMsgs
}

func (r *HostErrRecorder) FailedNodes() []string {
	r.lock.Lock()
	defer r.lock.Unlock()

	var nodes []string
	for k := range r.errMsgs {
		nodes = append(nodes, k)
	}

	return nodes
}
