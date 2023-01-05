package exception

import "fmt"

type UnsupportedRowTypeError struct {
	*baseError
	t string
}

func UnsupportedRowType(t interface{}) *UnsupportedRowTypeError {
	rowType := fmt.Sprintf("%T", t)
	message := fmt.Sprintf("Unsupported row type: %s", rowType)
	err := &UnsupportedRowTypeError{
		baseError: createBaseError(message),
		t:         rowType,
	}
	return err
}

func (e *UnsupportedRowTypeError) Type() string {
	return e.t
}

type RowLengthNotEqualColumnsError struct {
	*baseError
	rowLength    int
	columnLength int
}

func RowLengthNotEqualColumns(rowLength, columnLength int) *RowLengthNotEqualColumnsError {
	message := fmt.Sprintf("The length of row(%d) does not equal the columns(%d)", rowLength, columnLength)
	err := &RowLengthNotEqualColumnsError{
		baseError:    createBaseError(message),
		rowLength:    rowLength,
		columnLength: columnLength,
	}
	return err
}
