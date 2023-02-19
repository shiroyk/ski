package analyzer

import (
	"fmt"
	"runtime/debug"
	"sync/atomic"

	"github.com/shiroyk/cloudcat/parser"
	"github.com/shiroyk/cloudcat/schema"
	"golang.org/x/exp/slog"
)

var formatter atomic.Value

func init() {
	formatter.Store(new(defaultFormatHandler))
}

// SetDefaultFormatter set the default formatter
func SetDefaultFormatter(formatHandler FormatHandler) {
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
			ctx.Logger().Error(fmt.Sprintf("analyzer error %s", debug.Stack()), r.(error))
		}
	}()

	return analyze(ctx, s, content, "$")
}

func analyze(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string,
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

func analyzeString(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string,
) (result any) {
	var err error
	if s.Type == schema.ArrayType {
		result, err = s.Rule.GetStrings(ctx, content)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("analyze %s failed", path), err)
			return
		}
		ctx.Logger().Debug("parse", slog.String("path", path), slog.Any("result", result))
	} else {
		result, err = s.Rule.GetString(ctx, content)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("analyze %s failed", path), err)
			return
		}
		ctx.Logger().Debug("parse", slog.String("path", path), slog.Any("result", result))

		if s.Type != schema.StringType {
			result, err = GetFormatter().Format(result, s.Type)
			if err != nil {
				ctx.Logger().Error(fmt.Sprintf("format %s failed %v to %v",
					path, result, s.Format), err)
				return
			}
			ctx.Logger().Debug("format", slog.String("path", path), slog.Any("result", result))
		}
	}

	if s.Format != "" {
		result, err = GetFormatter().Format(result, s.Format)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("format %s failed %v to %v",
				path, result, s.Format), err)
			return
		}
		ctx.Logger().Debug("format", slog.String("path", path), slog.Any("result", result))
	}

	return
}

func analyzeObject(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string,
) (ret any) {
	if s.Properties != nil {
		element := analyzeInit(ctx, s, content, path)
		if len(element) == 0 {
			return
		}
		object := make(map[string]any, len(s.Properties))

		for field, s := range s.Properties {
			object[field] = analyze(ctx, &s, element[0], path+"."+field)
		}

		return object
	} else if s.Rule != nil {
		return analyzeString(ctx, s.CloneWithType(schema.ObjectType), content, path)
	}

	return
}

func analyzeArray(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string,
) any {
	if s.Properties != nil {
		elements := analyzeInit(ctx, s, content, path)
		array := make([]any, len(elements))

		for i, item := range elements {
			s := schema.NewSchema(schema.ObjectType).SetProperty(s.Properties)
			array[i] = analyzeObject(ctx, s, item, fmt.Sprintf("%s.[%v]", path, i))
		}

		return array
	} else if s.Rule != nil {
		return analyzeString(ctx, s.CloneWithType(schema.ArrayType), content, path)
	}

	return nil
}

func analyzeInit(
	ctx *parser.Context,
	s *schema.Schema,
	content any,
	path string,
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
		return elements
	}

	element, err := s.Init.GetElement(ctx, content)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("analyze %s init failed", path), err)
		return
	}
	return []string{element}
}
