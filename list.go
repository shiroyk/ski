package ski

import (
	"context"
	"errors"
)

// list_of exec slice of Executor.
// if Executor return ErrYield will be skipped
func list_of(args Arguments) (Executor, error) { return _list_of(args), nil }

type _list_of []Executor

func (l _list_of) Exec(ctx context.Context, arg any) (any, error) {
	ret := make([]any, 0, len(l))
	for _, exec := range l {
		v, err := exec.Exec(ctx, arg)
		if err != nil {
			if errors.Is(err, ErrYield) {
				return nil, nil
			}
			return nil, err
		}
		ret = append(ret, v)
	}
	return ret, nil
}
