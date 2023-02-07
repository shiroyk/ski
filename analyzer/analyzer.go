package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/shiroyk/cloudcat/parser"
	_ "github.com/shiroyk/cloudcat/parser/parsers/gq"
	_ "github.com/shiroyk/cloudcat/parser/parsers/js"
	_ "github.com/shiroyk/cloudcat/parser/parsers/json"
	_ "github.com/shiroyk/cloudcat/parser/parsers/regex"
	_ "github.com/shiroyk/cloudcat/parser/parsers/xpath"
	"github.com/spf13/cast"
)

type Analyzer struct {
	FormatHandler FormatHandler
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		FormatHandler: new(defaultFormatHandler),
	}
}

func (analyzer *Analyzer) ExecuteSchema(ctx *parser.Context, schema *parser.Schema, content string) any {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error("analyzer error ", r.(error))
		}
	}()

	return analyzer.process(ctx, schema, content)
}

func (analyzer *Analyzer) process(ctx *parser.Context, schema *parser.Schema, content any) any {
	switch schema.Type {
	default:
		return nil
	case parser.StringType, parser.IntegerType, parser.NumberType, parser.BooleanType:
		return analyzer.processString(ctx, schema, content)
	case parser.ObjectType:
		return analyzer.processObject(ctx, schema, content)
	case parser.ArrayType:
		return analyzer.processArray(ctx, schema, content)
	}
}

func (analyzer *Analyzer) processString(ctx *parser.Context, schema *parser.Schema, content any) any {
	var result any
	var err error
	if schema.Type == parser.ArrayType {
		result, err = schema.Rule.GetStrings(ctx, content)
		if err != nil {
			ctx.Logger().Error("process failed", err)
		}
	} else {
		result, err = schema.Rule.GetString(ctx, content)
		if err != nil {
			ctx.Logger().Error("process failed", err)
		}

		if schema.Type != parser.StringType {
			if result, err = analyzer.FormatHandler.Format(result, schema.Type); err != nil {
				ctx.Logger().Error("format failed", err)
			}
		}
	}

	if schema.Format != "" {
		if result, err = analyzer.FormatHandler.Format(result, schema.Format); err != nil {
			ctx.Logger().Error("format failed", err)
		}
	}

	return result
}

func (analyzer *Analyzer) processObject(ctx *parser.Context, schema *parser.Schema, content any) any {
	if schema.Properties != nil {
		element := analyzer.processInit(ctx, schema, content)[0]
		object := make(map[string]any, len(schema.Properties))

		for field, schema := range schema.Properties {
			object[field] = analyzer.process(ctx, &schema, element)
		}

		return object
	} else if schema.Rule != nil {
		return analyzer.processString(ctx, schema.CloneWithType(parser.ObjectType), content)
	}

	return nil
}

func (analyzer *Analyzer) processArray(ctx *parser.Context, schema *parser.Schema, content any) any {
	if schema.Properties != nil {
		elements := analyzer.processInit(ctx, schema, content)
		array := make([]any, len(elements))

		for i, item := range elements {
			s := parser.NewSchema(parser.ObjectType).SetProperty(schema.Properties)
			array[i] = analyzer.processObject(ctx, s, item)
		}

		return array
	} else if schema.Rule != nil {
		return analyzer.processString(ctx, schema.CloneWithType(parser.ArrayType), content)
	}

	return nil
}

func (analyzer *Analyzer) processInit(ctx *parser.Context, schema *parser.Schema, content any) []string {
	if schema.Init == nil || len(schema.Init) == 0 {
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

	if schema.Type == parser.ArrayType {
		elements, err := schema.Init.GetElements(ctx, content)
		if err != nil {
			ctx.Logger().Error("process init failed", err)
		}
		return elements
	}

	element, err := schema.Init.GetElement(ctx, content)
	if err != nil {
		ctx.Logger().Error("process init failed", err)
	}
	return []string{element}
}

// FormatHandler schema property formatter
type FormatHandler interface {
	// Format the data to the given parser.SchemaType
	Format(data any, format parser.SchemaType) (any, error)
}

type defaultFormatHandler struct{}

func (f defaultFormatHandler) Format(data any, format parser.SchemaType) (any, error) {
	switch data := data.(type) {
	case string:
		switch format {
		case parser.StringType:
			return data, nil
		case parser.IntegerType:
			return cast.ToIntE(data)
		case parser.NumberType:
			return cast.ToFloat64E(data)
		case parser.BooleanType:
			return cast.ToBoolE(data)
		case parser.ArrayType:
			slice := make([]any, 0)
			if err := json.Unmarshal([]byte(data), &slice); err != nil {
				return nil, err
			}
			return slice, nil
		case parser.ObjectType:
			object := make(map[string]any, 0)
			if err := json.Unmarshal([]byte(data), &object); err != nil {
				return nil, err
			}
			return object, nil
		}
	case []string:
		slice := make([]any, len(data))
		for i, o := range data {
			slice[i], _ = f.Format(o, format)
		}
		return slice, nil
	case map[string]any:
		maps := make(map[string]any, len(data))
		for k, v := range data {
			maps[k], _ = f.Format(v, format)
		}
		return maps, nil
	}
	return data, fmt.Errorf("unexpected type %T", data)
}
