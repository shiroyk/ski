package ski

import (
	"errors"
	"fmt"
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

// CompileError the compile error
type CompileError struct {
	line, column int
	msg          string
	err          error
}

// Pos the error position
func (e CompileError) Pos() (int, int) { return e.line, e.column }

func (e CompileError) Error() string {
	msg := e.msg
	if e.err != nil {
		msg = fmt.Sprintf("%s: %s", msg, e.err)
	}
	return fmt.Sprintf("line %d column %d %s", e.line, e.column, msg)
}

func (e CompileError) String() string { return e.Error() }

func (e CompileError) Unwrap() error { return e.err }

// ErrYield the yield control error
// it can be used to control the execution of the pipeline
// Executor support: list.of, pipe, each, mapping
var ErrYield = errorControl{yieldCode}
