package exception

type baseError struct {
	message string
}

func createBaseError(message string) *baseError {
	err := new(baseError)
	err.message = message
	return err
}

func (e *baseError) Error() string {
	return e.message
}

func (e *baseError) String() string {
	return e.Error()
}
