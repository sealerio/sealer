package util

import (
	"fmt"
)

// DeprecatedTips
// Arguments:
// - deleteFunction: delete function name
// - newFunction:    new function name
// - version:        support version
// - functionType:   method or function
func DeprecatedTips(
	deleteFunction,
	newFunction,
	version,
	functionType string) {
	fmt.Printf("%s `%s` will no longer supported."+
		" You can use the `%s` %s instead of `%s` %s."+
		" This %s will be removed in version %s.\n",
		Capitalize(functionType), deleteFunction, newFunction,
		functionType, deleteFunction, functionType, functionType, version,
	)
}
