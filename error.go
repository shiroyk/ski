package ski

import (
	"errors"
)

type code uint8

const (
	_ code = iota << 1
	yieldCode
)

type errorControl struct {
	code code
}

func (e errorControl) Error() string {
	switch e.code {
	case yieldCode:
		return "yield"
	default:
		return "unknown control code"
	}
}

func (e errorControl) String() string { return e.Error() }

func (e errorControl) Is(err error) bool {
	var ec errorControl
	ok := errors.As(err, &ec)
	if !ok {
		return false
	}
	return ec.code == e.code
}

// ErrYield the yield control error
// it can be used to control the execution of the pipeline
// Executor support: list.of, pipe, each, mapping
var ErrYield = errorControl{yieldCode}
