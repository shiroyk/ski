package cloudcat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync/atomic"

	"github.com/shiroyk/cloudcat/plugin"
	"github.com/spf13/cast"
)

var attr = slog.String("source", "analyze")

var defaultAnalyzer atomic.Value

func init() {
	defaultAnalyzer.Store(NewAnalyzer(new(defaultFormatHandler), true))
}

// SetAnalyzer sets the default Analyzer
func SetAnalyzer(analyzer Analyzer) {
	defaultAnalyzer.Store(analyzer)
}

// Analyze a Schema with default Analyzer, returns the result.
func Analyze(ctx *plugin.Context, schema *Schema, content string) any {
	return defaultAnalyzer.Load().(Analyzer).Analyze(ctx, schema, content)
}

// Analyzer the schema with content.
type Analyzer interface {
	// Analyze a Schema, returns the result.
	Analyze(ctx *plugin.Context, schema *Schema, content string) any
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(formatter FormatHandler, debug bool) Analyzer {
	return &analyzer{formatter, debug}
}

type analyzer struct {
	formatter FormatHandler
	debug     bool
}

// Analyze a Schema, returns the result
func (a *analyzer) Analyze(ctx *plugin.Context, schema *Schema, content string) any {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error(fmt.Sprintf("analyze error %s", r),
				"stack", string(debug.Stack()), attr)
		}
	}()
	return a.analyze(ctx, schema, content, "$")
}

// analyze execute the corresponding to analyze by schema.Type
func (a *analyzer) analyze(
	ctx *plugin.Context,
	schema *Schema,
	content any,
	path string, // the path of properties
) any {
	switch schema.Type {
	default:
		return nil
	case StringType, IntegerType, NumberType, BooleanType:
		return a.string(ctx, schema, content, path)
	case ObjectType:
		return a.object(ctx, schema, content, path)
	case ArrayType:
		return a.array(ctx, schema, content, path)
	}
}

// string get string or slice string and convert it to the specified type.
// If the type is not schema.StringType then convert to the specified type.
//
//nolint:nakedret
func (a *analyzer) string(
	ctx *plugin.Context,
	schema *Schema,
	content any,
	path string, // the path of properties
) (ret any) {
	var err error
	if schema.Type == ArrayType { //nolint:nestif
		ret, err = GetStrings(schema.Rule, ctx, content)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("failed analyze %s", path), "error", err, attr)
			return nil
		}
		if a.debug {
			ctx.Logger().Debug("parse", "path", path, "result", ret, attr)
		}
	} else {
		ret, err = GetString(schema.Rule, ctx, content)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("failed analyze %s", path), "error", err, attr)
			return nil
		}
		if a.debug {
			ctx.Logger().Debug("parse", "path", path, "result", ret, attr)
		}

		if schema.Type != StringType {
			ret, err = a.formatter.Format(ret, schema.Type)
			if err != nil {
				ctx.Logger().Error(fmt.Sprintf("failed format %s %v to %v",
					path, ret, schema.Type), "error", err, attr)
				return
			}
			if a.debug {
				ctx.Logger().Debug("format", "path", path, "result", ret, attr)
			}
		}
	}

	if schema.Format != "" {
		ret, err = a.formatter.Format(ret, schema.Format)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("failed format %s %v to %v",
				path, ret, schema.Format), "error", err, attr)
			return
		}
		if a.debug {
			ctx.Logger().Debug("format", "path", path, "result", ret, attr)
		}
	}

	return
}

// object get object.
// If properties is not empty, execute init to get the object element then analyze properties.
// If rule is not empty, execute string to get object.
func (a *analyzer) object(
	ctx *plugin.Context,
	schema *Schema,
	content any,
	path string, // the path of properties
) (ret any) {
	if schema.Properties != nil {
		var object map[string]any
		ks, k := schema.Properties["$key"]
		vs, v := schema.Properties["$value"]
		if k && v {
			elements := a.init(ctx, schema.Init, ArrayType, content, path)
			if len(elements) == 0 {
				return
			}
			object = make(map[string]any, len(elements))
			for i, element := range elements {
				key, err := GetString(ks.Rule, ctx, element)
				keyPath := fmt.Sprintf("%s.[%v].value", path, i)
				if a.debug {
					ctx.Logger().Debug("parse", "path", keyPath, "result", key, attr)
				}
				if err != nil {
					ctx.Logger().Error(fmt.Sprintf("failed to analyze key %v", keyPath), "error", err, attr)
					return nil
				}
				object[key] = a.analyze(ctx, &vs, element, keyPath)
			}
			return object
		}

		element := a.init(ctx, schema.Init, schema.Type, content, path)
		if len(element) == 0 {
			return
		}
		object = make(map[string]any, len(schema.Properties))

		for field, fieldSchema := range schema.Properties {
			object[field] = a.analyze(ctx, &fieldSchema, element[0], path+"."+field) //nolint:gosec
		}

		return object
	} else if schema.Rule != nil {
		return a.string(ctx, &Schema{
			Type:   ObjectType,
			Format: schema.Format,
			Rule:   schema.Rule,
		}, content, path)
	}

	return
}

// array get array.
// If properties is not empty, execute init to get the slice of elements then analyze properties.
// If rule is not empty, execute string to get array
func (a *analyzer) array(
	ctx *plugin.Context,
	s *Schema,
	content any,
	path string, // the path of properties
) any {
	if s.Properties != nil {
		elements := a.init(ctx, s.Init, s.Type, content, path)
		array := make([]any, len(elements))

		for i, item := range elements {
			newSchema := NewSchema(ObjectType).SetProperty(s.Properties)
			array[i] = a.object(ctx, newSchema, item, fmt.Sprintf("%s.[%v]", path, i))
		}

		return array
	} else if s.Rule != nil {
		return a.string(ctx, &Schema{
			Type:   ArrayType,
			Format: s.Format,
			Rule:   s.Rule,
		}, content, path)
	}

	return nil
}

// init get elements
func (a *analyzer) init(
	ctx *plugin.Context,
	init Action,
	typ Type,
	content any,
	path string, // the path of properties
) (ret []string) {
	if init == nil {
		switch data := content.(type) {
		case []string:
			return data
		case string:
			return []string{data}
		default:
			ctx.Logger().Error(fmt.Sprintf("failed analyze init %s", path),
				"error", fmt.Errorf("unexpected content type %T", content), attr)
			return
		}
	}

	if typ == ArrayType {
		elements, err := GetElements(init, ctx, content)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("failed analyze init %s", path), "error", err, attr)
			return
		}
		if a.debug {
			ctx.Logger().Debug("init", "path", path, "result", strings.Join(elements, "\n"), attr)
		}
		return elements
	}

	element, err := GetElement(init, ctx, content)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("failed analyze init %s", path), "error", err, attr)
		return
	}
	if a.debug {
		ctx.Logger().Debug("init", "path", path, "result", element, attr)
	}
	return []string{element}
}

// FormatHandler schema property formatter
type FormatHandler interface {
	// Format the data to the given Type
	Format(data any, format Type) (any, error)
}

type defaultFormatHandler struct{}

// Format the data to the given Type
func (f defaultFormatHandler) Format(data any, format Type) (ret any, err error) {
	switch data := data.(type) {
	case string:
		if data == "" && format != StringType {
			return
		}
		switch format {
		case StringType:
			return data, nil
		case IntegerType:
			ret, err = cast.ToIntE(data)
		case NumberType:
			ret, err = cast.ToFloat64E(data)
		case BooleanType:
			ret, err = cast.ToBoolE(data)
		case ArrayType:
			ret = make([]any, 0)
			err = json.Unmarshal([]byte(data), &ret)
		case ObjectType:
			ret = make(map[string]any, 0)
			err = json.Unmarshal([]byte(data), &ret)
		}
		if err != nil {
			return nil, err
		}
		return
	case []string:
		slice := make([]any, len(data))
		for i, o := range data {
			slice[i], err = f.Format(o, format)
			if err != nil {
				return nil, err
			}
		}
		return slice, nil
	case map[string]any:
		maps := make(map[string]any, len(data))
		for k, v := range data {
			maps[k], err = f.Format(v, format)
			if err != nil {
				return nil, err
			}
		}
		return maps, nil
	}
	return nil, fmt.Errorf("failed format type %T to %s", data, format)
}
