package analyzer

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetcher"
	"github.com/shiroyk/cloudcat/meta"
	"github.com/shiroyk/cloudcat/parser"
	_ "github.com/shiroyk/cloudcat/parser/parsers/gq"
	_ "github.com/shiroyk/cloudcat/parser/parsers/js"
	_ "github.com/shiroyk/cloudcat/parser/parsers/json"
	_ "github.com/shiroyk/cloudcat/parser/parsers/regex"
	_ "github.com/shiroyk/cloudcat/parser/parsers/xpath"
	"github.com/shiroyk/cloudcat/utils"
	"github.com/spf13/cast"
	"golang.org/x/exp/slog"
)

type Analyzer struct {
	FormatHandler FormatHandler
	fetcher       *fetcher.Fetcher
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		FormatHandler: new(defaultFormatHandler),
		fetcher:       di.MustResolve[*fetcher.Fetcher](),
	}
}

func (analyzer *Analyzer) ExecuteSchema(ctx *parser.Context, schema *meta.Schema, content any) any {
	defer analyzer.recoverMe()

	return analyzer.processSchema(ctx, schema, content)
}

func (analyzer *Analyzer) processSchema(ctx *parser.Context, schema *meta.Schema, content any) any {
	switch schema.Type {
	default:
		return nil
	case meta.StringType, meta.IntegerType, meta.NumberType, meta.BooleanType:
		return analyzer.processString(ctx, schema, content)
	case meta.ObjectType:
		return analyzer.processObject(ctx, schema, content)
	case meta.ArrayType:
		return analyzer.processArray(ctx, schema, content)
	}
}

func (analyzer *Analyzer) processString(ctx *parser.Context, schema *meta.Schema, content any) any {
	var result any
	var err error
	if schema.Type == meta.ArrayType {
		result = meta.ActEach(schema.Rule, content,
			func(each any, key string, arg string) ([]string, error) {
				if p, ok := parser.GetParser(key); ok {
					return p.GetStrings(ctx, each, arg)
				}
				return nil, errors.New("can not find parser")
			})
	} else {
		result = meta.ActEach(schema.Rule, content,
			func(each any, key string, arg string) (string, error) {
				if p, ok := parser.GetParser(key); ok {
					return p.GetString(ctx, each, arg)
				}
				return "", errors.New("can not find parser")
			})

		if schema.Type != meta.StringType {
			if result, err = analyzer.FormatHandler.Format(result, schema.Type); err != nil {
				panic(err)
			}
		}
	}

	if schema.Format != "" {
		if result, err = analyzer.FormatHandler.Format(result, schema.Format); err != nil {
			panic(err)
		}
	}

	return result
}

func (analyzer *Analyzer) processObject(ctx *parser.Context, schema *meta.Schema, content any) any {
	if schema.Properties != nil {
		genGroup := sync.WaitGroup{}
		buiGroup := sync.WaitGroup{}
		element := processInit(ctx, schema, content)[0]
		object := make(map[string]any, len(schema.Properties))
		objChan := make(chan utils.Pair[string, any], len(schema.Properties))

		buiGroup.Add(1)
		go func() {
			defer buiGroup.Done()
			for ele := range objChan {
				object[ele.Key] = ele.Value
			}
		}()

		for field, schema := range schema.Properties {
			genGroup.Add(1)
			go func(field string, schema meta.Schema) {
				defer genGroup.Done()
				objChan <- utils.Pair[string, any]{Key: field, Value: analyzer.processSchema(ctx, &schema, element)}
			}(field, schema)
		}

		genGroup.Wait()
		close(objChan)
		buiGroup.Wait()

		return object
	} else if schema.Rule != nil {
		return analyzer.processString(ctx, meta.NewSchema(meta.ObjectType, schema.Format).SetRule(schema.Rule...), content)
	} else {
		return nil
	}
}

func (analyzer *Analyzer) processArray(ctx *parser.Context, schema *meta.Schema, content any) any {
	if schema.Properties != nil {
		genGroup := sync.WaitGroup{}
		buiGroup := sync.WaitGroup{}
		elements := processInit(ctx, schema, content)
		array := make([]any, len(elements))
		arrayChan := make(chan utils.Pair[int, any], len(elements))

		buiGroup.Add(1)
		go func() {
			defer buiGroup.Done()
			for ele := range arrayChan {
				array[ele.Key] = ele.Value
			}
		}()

		for i, item := range elements {
			genGroup.Add(1)
			go func(i int, item any) {
				defer genGroup.Done()
				s := meta.NewSchema(meta.ObjectType).SetProperty(schema.Properties)
				arrayChan <- utils.Pair[int, any]{Key: i, Value: analyzer.processObject(ctx, s, item)}
			}(i, item)
		}

		genGroup.Wait()
		close(arrayChan)
		buiGroup.Wait()

		return array
	} else if schema.Rule != nil {
		return analyzer.processString(ctx, meta.NewSchema(meta.ArrayType, schema.Format).SetRule(schema.Rule...), content)
	} else {
		return nil
	}
}

func (analyzer *Analyzer) recoverMe() {
	if r := recover(); r != nil {
		slog.Error("analyzer error %s", r.(error))
	}
}

func processInit(ctx *parser.Context, schema *meta.Schema, content any) []string {
	if schema.Init == nil || len(schema.Init) == 0 {
		switch data := content.(type) {
		case []string, nil:
			return data.([]string)
		case string:
			return []string{data}
		default:
			panic(fmt.Errorf("unexpected content type %T", content))
		}
	} else {
		var elements []string
		if schema.Type == meta.ArrayType {
			elements = meta.ActEach(schema.Init, content,
				func(each any, key string, arg string) ([]string, error) {
					if p, ok := parser.GetParser(key); ok {
						return p.GetElements(ctx, each, arg)
					}
					return nil, errors.New("can not find parser")
				})
		} else {
			elements = []string{meta.ActEach(schema.Init, content,
				func(each any, key string, arg string) (string, error) {
					if p, ok := parser.GetParser(key); ok {
						return p.GetString(ctx, each, arg)
					}
					return "", errors.New("can not find parser")
				})}
		}

		return elements
	}
}

type FormatHandler interface {
	Format(data any, format meta.Type) (any, error)
}

type defaultFormatHandler struct{}

func (f defaultFormatHandler) Format(data any, format meta.Type) (any, error) {
	switch data := data.(type) {
	case string:
		switch format {
		case meta.StringType:
			return data, nil
		case meta.IntegerType:
			return cast.ToIntE(data)
		case meta.NumberType:
			return cast.ToFloat64E(data)
		case meta.BooleanType:
			return cast.ToBoolE(data)
		case meta.ArrayType:
			slice := make([]any, 0)
			if err := json.Unmarshal([]byte(data), &slice); err != nil {
				return nil, err
			}
			return slice, nil
		case meta.ObjectType:
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
