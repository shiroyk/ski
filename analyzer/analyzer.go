package analyzer

import (
	"fmt"
	"runtime/debug"
	"sync/atomic"

	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
)

var formatter atomic.Value

func init() {
	formatter.Store(new(defaultFormatHandler))
}

// SetFormatter set the formatter
func SetFormatter(formatHandler FormatHandler) {
	formatter.Store(formatHandler)
}

// GetFormatter get the formatter
func GetFormatter() FormatHandler {
	return formatter.Load().(FormatHandler)
}

// Analyze analyze a schema.Schema, returns the result
func Analyze(ctx *parser.Context, s *schema.Schema, content string) any {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error(fmt.Sprintf("analyze error %s", r), nil,
				"stack", string(debug.Stack()))
		}
	}()

	return analyze(ctx, s, content, "$")
}

// analyze execute the corresponding to analyze by schema.Type
func analyze(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string, // the path of properties
) any {
	switch s.Type {
	default:
		return nil
	case schema.StringType, schema.IntegerType, schema.NumberType, schema.BooleanType:
		return analyzeString(ctx, s, content, path)
	case schema.ObjectType:
		return analyzeObject(ctx, s, content, path)
	case schema.ArrayType:
		return analyzeArray(ctx, s, content, path)
	}
}

// analyzeString get string or slice string and convert it to the specified type.
// If the type is not schema.StringType then convert to the specified type.
//
//nolint:nakedret
func analyzeString(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string, // the path of properties
) (ret any) {
	var err error
	if s.Type == schema.ArrayType { //nolint:nestif
		ret, err = s.Rule.GetStrings(ctx, content)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("analyze %s failed", path), err)
			return
		}
		ctx.Logger().Debug("parse", "path", path, "result", ret)
	} else {
		ret, err = s.Rule.GetString(ctx, content)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("analyze %s failed", path), err)
			return
		}
		ctx.Logger().Debug("parse", "path", path, "result", ret)

		if s.Type != schema.StringType {
			ret, err = GetFormatter().Format(ret, s.Type)
			if err != nil {
				ctx.Logger().Error(fmt.Sprintf("format %s failed %v to %v",
					path, ret, s.Format), err)
				return
			}
			ctx.Logger().Debug("format", "path", path, "result", ret)
		}
	}

	if s.Format != "" {
		ret, err = GetFormatter().Format(ret, s.Format)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("format %s failed %v to %v",
				path, ret, s.Format), err)
			return
		}
		ctx.Logger().Debug("format", "path", path, "result", ret)
	}

	return
}

// analyzeObject get object.
// If properties is not empty, execute analyzeInit to get the object element then analyze properties.
// If rule is not empty, execute analyzeString to get object.
func analyzeObject(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string, // the path of properties
) (ret any) {
	if s.Properties != nil {
		element := analyzeInit(ctx, s, content, path)
		if len(element) == 0 {
			return
		}
		object := make(map[string]any, len(s.Properties))

		for field, fieldSchema := range s.Properties {
			object[field] = analyze(ctx, &fieldSchema, element[0], path+"."+field) //nolint:gosec
		}

		return object
	} else if s.Rule != nil {
		return analyzeString(ctx, s.CloneWithType(schema.ObjectType), content, path)
	}

	return
}

// analyzeArray get array.
// If properties is not empty, execute analyzeInit to get the slice of elements then analyze properties.
// If rule is not empty, execute analyzeString to get array
func analyzeArray(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string, // the path of properties
) any {
	if s.Properties != nil {
		elements := analyzeInit(ctx, s, content, path)
		array := make([]any, len(elements))

		for i, item := range elements {
			newSchema := schema.NewSchema(schema.ObjectType).SetProperty(s.Properties)
			array[i] = analyzeObject(ctx, newSchema, item, fmt.Sprintf("%s.[%v]", path, i))
		}

		return array
	} else if s.Rule != nil {
		return analyzeString(ctx, s.CloneWithType(schema.ArrayType), content, path)
	}

	return nil
}

// analyzeInit get type of object or array elements
func analyzeInit(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string, // the path of properties
) (ret []string) {
	if len(s.Init) == 0 {
		switch data := content.(type) {
		case []string:
			return data
		case string:
			return []string{data}
		default:
			ctx.Logger().Error(fmt.Sprintf("analyze %s init failed", path),
				fmt.Errorf("unexpected content type %T", content))
			return
		}
	}

	if s.Type == schema.ArrayType {
		elements, err := s.Init.GetElements(ctx, content)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("analyze %s init failed", path), err)
			return
		}
		ctx.Logger().Debug("init", "path", path, "result", len(elements))
		return elements
	}

	element, err := s.Init.GetElement(ctx, content)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("analyze %s init failed", path), err)
		return
	}
	ctx.Logger().Debug("init", "path", path, "result", 1)
	return []string{element}
}
