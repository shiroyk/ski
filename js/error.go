package js

import (
	"errors"
	"unsafe"
)

type joinError []error

func (e joinError) Error() string {
	// Since Join returns nil if every value in errs is nil,
	// e.errs cannot be empty.
	if len(e) == 1 {
		return e[0].Error()
	}

	b := []byte(e[0].Error())
	for _, err := range e[1:] {
		b = append(b, '\n')
		b = append(b, err.Error()...)
	}
	// At this point, b has at least one byte '\n'.
	return unsafe.String(&b[0], len(b))
}

func (e joinError) Unwrap() []error { return e }

var errCallableDefault = errors.New("module default export is not a function")
