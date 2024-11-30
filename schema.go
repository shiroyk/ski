package ski

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

func init() {
	Registers(NewExecutors{
		"fetch":       fetch,
		"raw":         raw,
		"kind":        kind,
		"map":         mapping,
		"each":        each,
		"pipe":        pipe,
		"or":          or,
		"debug":       debug,
		"list.of":     list_of,
		"if.contains": if_contains,
		"str.join":    str_join,
		"str.split":   str_split,
		"str.suffix":  str_suffix,
		"str.prefix":  str_prefix,
		"json.parse":  json_parse,
		"json.string": json_string,
	})
}

// Meta from the yaml node, if Executor implements Meta, it will be called on compile
type Meta interface {
	Meta(k, v *yaml.Node) error
}

// Compile the Executor from the YAML string.
func Compile(str string) (Executor, error) {
	c := new(compiler)
	if err := yaml.Unmarshal([]byte(str), c); err != nil {
		return nil, err
	}
	return c.exec, nil
}

// CompileNode the Executor from the YAML node.
func CompileNode(node *yaml.Node) (Executor, error) {
	c := new(compiler)
	if err := c.UnmarshalYAML(node); err != nil {
		return nil, err
	}
	return c.exec, nil
}

// Kind converts to a Kind type
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

// kind converts to a Kind type
func kind(args Arguments) (Executor, error) {
	var k Kind
	if err := k.UnmarshalText([]byte(args.GetString(0))); err != nil {
		return nil, err
	}
	return k, nil
}

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

type compiler struct {
	exec Executor
}

func (c compiler) newError(message string, node *yaml.Node, err error) error {
	if err != nil {
		message = fmt.Sprintf("%s: %s", message, err)
	}
	return fmt.Errorf("line %d column %d %s", node.Line, node.Column, message)
}

// UnmarshalYAML compile the Executor from the YAML string.
func (c *compiler) UnmarshalYAML(node *yaml.Node) error {
	exec, err := c.compileNode(node)
	if err != nil {
		return err
	}
	switch len(exec) {
	case 1:
		c.exec = exec[0]
	default:
		c.exec = c.piping(exec)
	}
	return nil
}

// piping return the first arg if the length is 1, else return Pipe
func (c compiler) piping(args []Executor) Executor {
	if len(args) == 1 {
		return args[0]
	}
	return Pipe(args)
}

// compileExecutor return the Executor with the key and values
func (c compiler) compileExecutor(k, v *yaml.Node) (Executor, error) {
	key := strings.TrimPrefix(k.Value, "$")
	init, ok := GetExecutor(key)
	if !ok {
		return nil, c.newError("executor not found", k, errors.New(key))
	}
	args, err := c.compileNode(v)
	if err != nil {
		return nil, err
	}
	exec, err := init(args)
	if err != nil {
		return nil, c.newError(key, k, err)
	}
	if meta, ok := exec.(Meta); ok {
		if err = meta.Meta(k, v); err != nil {
			return nil, c.newError("meta", k, err)
		}
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
		if node.Tag == "!!null" {
			return []Executor{_raw{nil}}, nil
		}
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

		pipe := make(Pipe, 0, len(valueNode.Content)/2)
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

type _map []Executor

// mapping return a map with the key and values [k1, v1, k2, v2, ...]
// if the key Executor implements If and condition not met, it will be skipped
func mapping(args Arguments) (Executor, error) {
	m := _map(args)
	if len(m)%2 != 0 {
		m = append(m, Raw(nil))
	}
	return m, nil
}

func (m _map) Exec(ctx context.Context, arg any) (any, error) {
	var ret map[string]any

	exec := func(arg any) {
		for i := 0; i < len(m); i += 2 {
			k, err := m[i].Exec(ctx, arg)
			if err != nil {
				continue
			}
			key, err := cast.ToStringE(k)
			if err != nil {
				continue
			}
			value, _ := m[i+1].Exec(ctx, arg)
			ret[key] = value
		}
	}

	v := reflect.ValueOf(arg)
	switch v.Kind() {
	case reflect.Slice:
		ret = make(map[string]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			exec(v.Index(i).Interface())
		}
		return ret, nil
	default:
		ret = make(map[string]any, len(m)/2)
		exec(arg)
		return ret, nil
	}
}

type _each struct{ Executor }

// each loop the slice arg and execute the Executor,
// if Executor return ErrYield will be skipped.
func each(args Arguments) (Executor, error) {
	if len(args) != 1 {
		return nil, errors.New("each needs 1 parameter")
	}
	return _each{args.Get(0)}, nil
}

func (each _each) Exec(ctx context.Context, arg any) (any, error) {
	v := reflect.ValueOf(arg)
	switch v.Kind() {
	case reflect.Slice:
		ret := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			v, err := each.Executor.Exec(ctx, v.Index(i).Interface())
			if err != nil {
				if errors.Is(err, ErrYield) {
					continue
				}
				return nil, err
			}
			ret = append(ret, v)
		}
		return ret, nil
	default:
		ret, err := each.Executor.Exec(ctx, arg)
		if err != nil {
			return nil, err
		}
		return ret, nil
	}
}

// Pipe executes a slice of Executor.
// if the Executor implements If and condition not met, it will be skipped.
type Pipe []Executor

// pipe executes a slice of Executor.
// if Executor return ErrYield will be stopped
func pipe(args Arguments) (Executor, error) { return Pipe(args), nil }

func (pipe Pipe) Exec(ctx context.Context, arg any) (ret any, err error) {
	switch len(pipe) {
	case 0:
		return nil, nil
	case 1:
		return pipe[0].Exec(ctx, arg)
	default:
		ret = arg

		for _, exec := range pipe {
			ret, err = exec.Exec(ctx, ret)
			if err != nil || ret == nil {
				return
			}
		}

		return
	}
}

// or executes a slice of Executor. return result if the Executor result is not nil.
func or(args Arguments) (Executor, error) { return _or(args), nil }

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

// Raw the Executor for raw value, return the original value
func Raw(arg any) Executor { return _raw{arg} }

// Raw the Executor for raw value, return the original value
func raw(args Arguments) (Executor, error) { return args.Get(0), nil }

type _raw struct{ any }

func (raw _raw) Exec(context.Context, any) (any, error) { return raw.any, nil }

// debug output the debug message and the origin arg
func debug(args Arguments) (Executor, error) {
	return _debug(args.GetString(0)), nil
}

type _debug string

func (debug _debug) Exec(ctx context.Context, v any) (any, error) {
	Logger(ctx).LogAttrs(ctx, slog.LevelDebug, string(debug), slog.Any("value", v))
	return v, nil
}
