package fluent

type Error struct {
	Err error
}

func (e Error) Error() string {
	return e.Err.Error()
}

// Recover invokes a function within a panic-recovering context, and returns
// any raised fluent.Error values; any other values are re-panicked.
//
// This can be useful for writing large blocks of code using fluent nodes,
// and handling any errors at once at the end.
func Recover(fn func()) (err error) {
	defer func() {
		ei := recover()
		switch e2 := ei.(type) {
		case nil:
			return
		case Error:
			err = e2
		default:
			panic(ei)
		}
	}()
	fn()
	return
}
