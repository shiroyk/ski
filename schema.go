package ski

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"

	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

type Kind uint

const (
	KindAny Kind = iota
	KindBool
	KindInt // int32
	KindInt64
	KindFloat // float 32
	KindFloat64
	KindString
)

var kindNames = [...]string{
	KindAny:     "any",
	KindBool:    "bool",
	KindInt:     "int",
	KindInt64:   "int64",
	KindFloat:   "float",
	KindFloat64: "float64",
	KindString:  "string",
}

func (k Kind) String() string { return kindNames[k] }

func (k Kind) MarshalText() (text []byte, err error) { return []byte(kindNames[k]), nil }

func (k *Kind) UnmarshalText(text []byte) error {
	switch string(text) {
	case "", "any":
		*k = KindAny
	case "bool":
		*k = KindBool
	case "int", "int32":
		*k = KindInt
	case "int64":
		*k = KindInt64
	case "float", "float32":
		*k = KindFloat
	case "float64":
		*k = KindFloat64
	case "string":
		*k = KindString
	default:
		return fmt.Errorf("unknown kind %s", text)
	}
	return nil
}

func (k Kind) Exec(_ context.Context, v any) (any, error) {
	switch k {
	case KindBool:
		return cast.ToBoolE(v)
	case KindInt:
		return cast.ToInt32E(v)
	case KindInt64:
		return cast.ToInt64E(v)
	case KindFloat:
		return cast.ToFloat32E(v)
	case KindFloat64:
		return cast.ToFloat64E(v)
	case KindString:
		return cast.ToStringE(v)
	default:
		return v, nil
	}
}

type (
	// Executor accept the argument and output result
	Executor interface {
		Exec(context.Context, any) (any, error)
	}

	// ExecutorMap map of the Executor init function
	ExecutorMap map[string]func(args ...Executor) (Executor, error)
)

type compiler struct {
	funcs ExecutorMap
	meta  func(node *yaml.Node, exec Executor, isParser bool) Executor
}

func (c compiler) newError(message string, node *yaml.Node, err error) error {
	if err != nil {
		message = fmt.Sprintf("%s: %s", message, err)
	}
	return fmt.Errorf("line %d column %d %s", node.Line, node.Column, message)
}

// compile the Executor from the YAML string.
func (c compiler) compile(str string) (Executor, error) {
	node := new(yaml.Node)
	if err := yaml.Unmarshal([]byte(str), node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %s", err)
	}
	if node.Kind != yaml.DocumentNode || len(node.Content) != 1 {
		return nil, errors.New("invalid YAML schema: document node is missing or incorrect")
	}
	exec, err := c.compileNode(node.Content[0])
	if err != nil {
		return nil, err
	}
	if len(exec) == 1 {
		return exec[0], nil
	}
	return c.piping(exec), nil
}

// piping return the first arg if the length is 1, else return _pipe
func (c compiler) piping(args []Executor) Executor {
	if len(args) == 1 {
		return args[0]
	}
	return _pipe(args)
}

// compileExecutor return the Executor with the key and values
func (c compiler) compileExecutor(k, v *yaml.Node) (Executor, error) {
	key := strings.TrimPrefix(k.Value, "$")
	init, ok := c.funcs[key]
	if ok {
		args, err := c.compileNode(v)
		if err != nil {
			return nil, err
		}
		exec, err := init(args...)
		if err != nil {
			return nil, c.newError(key, k, err)
		}
		if c.meta != nil {
			return c.meta(k, exec, false), nil
		}
		return exec, nil
	}

	name, method, found := strings.Cut(key, ".")
	parser, ok := GetParser(name)
	if !ok {
		return nil, c.newError("function or parser not found", k, errors.New(key))
	}

	var (
		exec Executor
		err  error
	)

	if found && method != "value" {
		parser, ok := parser.(ElementParser)
		if !ok {
			return nil, c.newError("method not found", k, errors.New(key))
		}
		switch method {
		case "element":
			exec, err = parser.Element(v.Value)
		case "elements":
			exec, err = parser.Elements(v.Value)
		default:
			return nil, c.newError("method not found", k, errors.New(key))
		}
	} else {
		exec, err = parser.Value(v.Value)
	}

	if err != nil {
		return nil, c.newError(key, k, err)
	}
	if c.meta != nil {
		return c.meta(k, exec, true), nil
	}
	return exec, nil
}

func (c compiler) compileNode(node *yaml.Node) ([]Executor, error) {
	switch node.Kind {
	case yaml.MappingNode:
		return c.compileMapping(node)
	case yaml.SequenceNode:
		return c.compileSequence(node)
	case yaml.ScalarNode:
		return []Executor{String(node.Value)}, nil
	case yaml.AliasNode:
		return c.compileNode(node.Alias)
	default:
		return nil, c.newError("invalid node type", node, nil)
	}
}

func (c compiler) compileSequence(node *yaml.Node) ([]Executor, error) {
	args := make([]Executor, 0, len(node.Content))
	for _, item := range node.Content {
		items, err := c.compileNode(item)
		if err != nil {
			return nil, err
		}
		args = append(args, c.piping(items))
	}
	return args, nil
}

func (c compiler) compileMapping(node *yaml.Node) ([]Executor, error) {
	if len(node.Content) == 0 || len(node.Content)%2 != 0 {
		return nil, c.newError("mapping node requires at least two elements", node, nil)
	}

	if strings.HasPrefix(node.Content[0].Value, "$") {
		ret := make([]Executor, 0, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			exec, err := c.compileExecutor(node.Content[i], node.Content[i+1])
			if err != nil {
				return nil, err
			}
			ret = append(ret, exec)
		}
		return ret, nil
	}

	ret := make([]Executor, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keyNode, valueNode := node.Content[i], node.Content[i+1]
		key := String(keyNode.Value)

		if valueNode.Kind != yaml.MappingNode {
			child, err := c.compileNode(valueNode)
			if err != nil {
				return nil, err
			}
			ret = append(ret, key, c.piping(child))
			continue
		}

		if len(valueNode.Content) == 2 {
			exec, err := c.compileExecutor(valueNode.Content[0], valueNode.Content[1])
			if err != nil {
				return nil, err
			}
			ret = append(ret, key, exec)
			continue
		}

		pipe := make(_pipe, 0, len(valueNode.Content)/2)
		for j := 0; j < len(valueNode.Content); j += 2 {
			exec, err := c.compileExecutor(valueNode.Content[j], valueNode.Content[j+1])
			if err != nil {
				return nil, err
			}
			pipe = append(pipe, exec)
		}
		ret = append(ret, key, pipe)
	}
	return ret, nil
}

type Option func(*compiler)

// WithExecutorMap external ExecutorMap
func WithExecutorMap(fn ExecutorMap) Option {
	return func(parser *compiler) {
		funcs := maps.Clone(buildInExecutor)
		maps.Copy(funcs, fn)
		parser.funcs = funcs
	}
}

type Meta = func(node *yaml.Node, exec Executor, isParser bool) Executor

// WithMeta with the meta message
func WithMeta(meta Meta) Option {
	return func(parser *compiler) { parser.meta = meta }
}

// Compile the Executor with the Option.
func Compile(str string, opts ...Option) (Executor, error) {
	c := new(compiler)
	for _, opt := range opts {
		opt(c)
	}
	if c.funcs == nil {
		c.funcs = buildInExecutor
	}
	return c.compile(str)
}

var buildInExecutor = ExecutorMap{
	"debug": func(args ...Executor) (Executor, error) {
		if len(args) > 0 {
			return _debug(ToString(args[0])), nil
		}
		return _debug(""), nil
	},
	"kind": func(args ...Executor) (Executor, error) {
		if len(args) != 1 {
			return nil, errors.New("kind needs 1 parameter")
		}
		var k Kind
		return k, k.UnmarshalText([]byte(ToString(args[0])))
	},
	"each": func(args ...Executor) (Executor, error) {
		if len(args) != 1 {
			return nil, errors.New("each needs 1 parameter")
		}
		return _each{args[0]}, nil
	},
	"json.parse":  func(args ...Executor) (Executor, error) { return _jsonParse{}, nil },
	"json.string": func(args ...Executor) (Executor, error) { return _jsonString{}, nil },
	"map": func(args ...Executor) (Executor, error) {
		kv := _map(args)
		if len(kv)%2 != 0 {
			kv = append(kv, Raw(nil))
		}
		return kv, nil
	},
	"or": func(args ...Executor) (Executor, error) { return _or(args), nil },
	"string.join": func(args ...Executor) (Executor, error) {
		if len(args) > 0 {
			return _stringJoin(ToString(args[0])), nil
		}
		return _stringJoin(""), nil
	},
	"pipe": func(args ...Executor) (Executor, error) { return _pipe(args), nil },
}

// String the Executor for string value
type String string

func (k String) Exec(_ context.Context, _ any) (any, error) { return k.String(), nil }

func (k String) String() string { return string(k) }

type _map []Executor

func (m _map) Exec(ctx context.Context, arg any) (any, error) {
	var ret map[string]any

	exec := func(a any) {
		for i := 0; i < len(m); i += 2 {
			k, err := m[i].Exec(ctx, a)
			if err != nil {
				continue
			}
			ks, err := cast.ToStringE(k)
			if err != nil {
				continue
			}
			v, _ := m[i+1].Exec(ctx, a)
			ret[ks] = v
		}
	}

	switch s := arg.(type) {
	case []any:
		ret = make(map[string]any, len(s))
		for _, a := range s {
			exec(a)
		}
		return ret, nil
	case []string:
		ret = make(map[string]any, len(s))
		for _, a := range s {
			exec(a)
		}
		return ret, nil
	default:
		ret = make(map[string]any, len(m)/2)
		exec(arg)
		return ret, nil
	}
}

type _each struct{ Executor }

func (each _each) Exec(ctx context.Context, arg any) (any, error) {
	switch s := arg.(type) {
	case []any:
		ret := make([]any, 0, len(s))
		for _, i := range s {
			v, _ := each.Executor.Exec(ctx, i)
			ret = append(ret, v)
		}
		return ret, nil
	case []string:
		ret := make([]any, 0, len(s))
		for _, i := range s {
			v, _ := each.Executor.Exec(ctx, i)
			ret = append(ret, v)
		}
		return ret, nil
	default:
		v, err := each.Executor.Exec(ctx, arg)
		if err != nil {
			return []any{}, nil
		}
		return []any{v}, nil
	}
}

// Raw the Executor for raw value, return the original value
func Raw(arg any) Executor { return _raw{arg} }

type _raw struct{ any }

func (raw _raw) Exec(context.Context, any) (any, error) { return raw.any, nil }

type _pipe []Executor

func (pipe _pipe) Exec(ctx context.Context, v any) (any, error) {
	switch len(pipe) {
	case 0:
		return nil, nil
	case 1:
		return pipe[0].Exec(ctx, v)
	default:
		ret, err := pipe[0].Exec(ctx, v)
		if err != nil {
			return nil, err
		}
		for _, s := range pipe[1:] {
			ret, err = s.Exec(ctx, ret)
			if err != nil {
				return nil, err
			}
		}
		return ret, nil
	}
}

type _or []Executor

func (or _or) Exec(ctx context.Context, arg any) (any, error) {
	for _, exec := range or {
		v, err := exec.Exec(ctx, arg)
		if err != nil {
			continue
		}
		if v != nil {
			return v, nil
		}
	}
	return nil, nil
}

type _debug string

func (debug _debug) Exec(ctx context.Context, v any) (any, error) {
	Logger(ctx).LogAttrs(ctx, slog.LevelDebug, string(debug), slog.Any("value", v))
	return v, nil
}

type _stringJoin string

func (sep _stringJoin) Exec(_ context.Context, arg any) (any, error) {
	switch s := arg.(type) {
	case []any:
		str, err := cast.ToStringSliceE(s)
		if err != nil {
			return nil, fmt.Errorf("expected string or []string, but got type %T", arg)
		}
		return strings.Join(str, string(sep)), nil
	case []string:
		return strings.Join(s, string(sep)), nil
	case string:
		return s, nil
	default:
		return nil, fmt.Errorf("expected string or []string, but got type %T", arg)
	}
}

type _jsonParse struct{}

func (_jsonParse) Exec(_ context.Context, v any) (any, error) {
	s, err := cast.ToStringE(v)
	if err != nil {
		return nil, err
	}
	var ret any
	err = json.Unmarshal([]byte(s), &ret)
	return ret, err
}

type _jsonString struct{}

func (_jsonString) Exec(_ context.Context, v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}
