package exception

import "fmt"

type ColumnsLengthError struct {
	*baseError
}

func ColumnsLength() *ColumnsLengthError {
	err := &ColumnsLengthError{createBaseError("columns length must more than zero")}
	return err
}

type ColumnDoNotExistError struct {
	*baseError
	name string
}

func (e *ColumnDoNotExistError) Name() string {
	return e.name
}

func ColumnDoNotExist(name string) *ColumnDoNotExistError {
	message := fmt.Sprintf("column %s do not exist", name)
	err := &ColumnDoNotExistError{createBaseError(message), name}
	return err
}
