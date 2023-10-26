package dsindex

import "errors"

var (
	ErrEmptyKey   = errors.New("key is empty")
	ErrEmptyValue = errors.New("value is empty")
)
