package ski

import (
	"context"
)

// list_of exec slice of Executor, if the Executor implements If
// and condition not met, it will be skipped.
func list_of(args Arguments) (Executor, error) { return _list_of(args), nil }

type _list_of []Executor

func (l _list_of) Exec(ctx context.Context, arg any) (any, error) {
	ret := make([]any, 0, len(l))
	for _, exec := range l {
		if control, ok := exec.(If); ok && !control.If(ctx, arg) {
			continue
		}
		v, err := exec.Exec(ctx, arg)
		if err != nil {
			return nil, err
		}
		ret = append(ret, v)
	}
	return ret, nil
}
