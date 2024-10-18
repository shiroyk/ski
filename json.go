package ski

import (
	"context"
	"encoding/json"

	"github.com/spf13/cast"
)

// json_parse unmarshal the argument as JSON
func json_parse(_ Arguments) (Executor, error) { return _json_parse{}, nil }

type _json_parse struct{}

func (_json_parse) Exec(_ context.Context, v any) (any, error) {
	s, err := cast.ToStringE(v)
	if err != nil {
		return nil, err
	}
	var ret any
	err = json.Unmarshal([]byte(s), &ret)
	return ret, err
}

// json_string marshal the argument as JSON string
func json_string(_ Arguments) (Executor, error) { return _json_string{}, nil }

type _json_string struct{}

func (_json_string) Exec(_ context.Context, v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}
