/*
Copyright © 2021 Alibaba Group Holding Ltd.
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"context"
	"io"
)

type Cmd interface {
	Run() error
	SetEnv(...string) Cmd
	SetStdin(io.Reader) Cmd
	SetStdout(io.Writer) Cmd
	SetStderr(io.Writer) Cmd
}

type Cmder interface {
	Command(string, ...string) Cmd
	CommandContext(context.Context, string, ...string) Cmd
}
