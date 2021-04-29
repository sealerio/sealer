package progress

type Task interface {
	ContextService
}

type TakeOverTask struct {
	Action func(cxt Context) error
	Cxt    Context
}

type ChannelTask struct {
	ProgressChan chan Msg
}

type successMsgTask struct {
}

func (c ChannelTask) context() Context {
	return Context{}
}

func (t TakeOverTask) context() Context {
	return t.Cxt
}

func (s successMsgTask) context() Context {
	return Context{}
}
