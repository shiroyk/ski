package analyzer

import (
	"fmt"
	"runtime/debug"
	"sync/atomic"

	_ "github.com/shiroyk/cloudcat/js"
	"github.com/shiroyk/cloudcat/parser"
	_ "github.com/shiroyk/cloudcat/parser/parsers/gq"
	_ "github.com/shiroyk/cloudcat/parser/parsers/js"
	_ "github.com/shiroyk/cloudcat/parser/parsers/json"
	_ "github.com/shiroyk/cloudcat/parser/parsers/regex"
	_ "github.com/shiroyk/cloudcat/parser/parsers/xpath"
	"github.com/shiroyk/cloudcat/schema"
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
			ctx.Logger().Error("analyzer error %v", r.(error), debug.Stack())
		}
	}()

	return process(ctx, s, content)
}

func process(ctx *parser.Context, s *schema.Schema, content any) any {
	switch s.Type {
	default:
		return nil
	case schema.StringType, schema.IntegerType, schema.NumberType, schema.BooleanType:
		return processString(ctx, s, content)
	case schema.ObjectType:
		return processObject(ctx, s, content)
	case schema.ArrayType:
		return processArray(ctx, s, content)
	}
}

func processString(ctx *parser.Context, s *schema.Schema, content any) any {
	var result any
	var err error
	if s.Type == schema.ArrayType {
		result, err = s.Rule.GetStrings(ctx, content)
		if err != nil {
			ctx.Logger().Error("process failed", err)
		}
	} else {
		result, err = s.Rule.GetString(ctx, content)
		if err != nil {
			ctx.Logger().Error("process failed", err)
		}

		if s.Type != schema.StringType {
			result, err = GetFormatter().Format(result, s.Type)
			if err != nil {
				ctx.Logger().Error(fmt.Sprintf("format failed %v to %v", result, s.Format), err)
			}
		}
	}

	if s.Format != "" {
		result, err = GetFormatter().Format(result, s.Format)
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("format failed %v to %v", result, s.Format), err)
		}
	}

	return result
}

func processObject(ctx *parser.Context, s *schema.Schema, content any) any {
	if s.Properties != nil {
		element := processInit(ctx, s, content)[0]
		object := make(map[string]any, len(s.Properties))

		for field, s := range s.Properties {
			object[field] = process(ctx, &s, element)
		}

		return object
	} else if s.Rule != nil {
		return processString(ctx, s.CloneWithType(schema.ObjectType), content)
	}

	return nil
}

func processArray(ctx *parser.Context, s *schema.Schema, content any) any {
	if s.Properties != nil {
		elements := processInit(ctx, s, content)
		array := make([]any, len(elements))

		for i, item := range elements {
			s := schema.NewSchema(schema.ObjectType).SetProperty(s.Properties)
			array[i] = processObject(ctx, s, item)
		}

		return array
	} else if s.Rule != nil {
		return processString(ctx, s.CloneWithType(schema.ArrayType), content)
	}

	return nil
}

func processInit(ctx *parser.Context, s *schema.Schema, content any) []string {
	if s.Init == nil || len(s.Init) == 0 {
		switch data := content.(type) {
		case []string, nil:
			return data.([]string)
		case string:
			return []string{data}
		default:
			ctx.Logger().Error("process init failed", fmt.Errorf("unexpected content type %T", content))
			return nil
		}
	}

	if s.Type == schema.ArrayType {
		elements, err := s.Init.GetElements(ctx, content)
		if err != nil {
			ctx.Logger().Error("process init failed", err)
		}
		return elements
	}

	element, err := s.Init.GetElement(ctx, content)
	if err != nil {
		ctx.Logger().Error("process init failed", err)
	}
	return []string{element}
}
