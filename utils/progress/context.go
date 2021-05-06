package progress

import (
	"io"

	"github.com/vbauerster/mpb/v6"
)

const (
	ReaderClose       = "ReaderClose"
	CurrentProcessBar = "ProcessBar"
)

type ContextService interface {
	context() Context
}

type Context map[string]interface{}

type processJob struct {
	function func(cxt Context) error
	cxt      Context
}

func (cxt Context) WithReader(reader io.Reader) Context {
	//var rc io.ReadCloser
	//rc, ok := reader.(io.ReadCloser)
	//if !ok {
	//	rc = io.NopCloser(reader)
	//}
	cxt[ReaderClose] = reader
	return cxt
}

func (cxt Context) CopyAllVar(srcCxt Context) {
	for k, v := range srcCxt {
		cxt[k] = v
	}
}

func (cxt Context) WithCurrentProcessBar(bar *mpb.Bar) Context {
	cxt[CurrentProcessBar] = bar
	return cxt
}

// only can retrieve once
func (cxt Context) GetCurrentReaderCloser() io.ReadCloser {
	rc, ok := cxt[ReaderClose]
	if !ok {
		return nil
	}
	rrc, ok := rc.(io.ReadCloser)
	if !ok {
		return nil
	}
	delete(cxt, ReaderClose)
	return rrc
}

func (cxt Context) GetCurrentBar() *mpb.Bar {
	rc, ok := cxt[CurrentProcessBar]
	if !ok {
		return nil
	}
	rrc, ok := rc.(*mpb.Bar)
	if !ok {
		return nil
	}
	return rrc
}
