package analyzer

import (
	"encoding/json"
	"fmt"

	"github.com/shiroyk/cloudcat/schema"
	"github.com/spf13/cast"
)

// FormatHandler schema property formatter
type FormatHandler interface {
	// Format the data to the given schema.SchemaType
	Format(data any, format schema.Type) (any, error)
}

type defaultFormatHandler struct{}

// Format the data to the given schema.SchemaType
func (f defaultFormatHandler) Format(data any, format schema.Type) (any, error) {
	switch data := data.(type) {
	case string:
		switch format {
		case schema.StringType:
			return data, nil
		case schema.IntegerType:
			return cast.ToIntE(data)
		case schema.NumberType:
			return cast.ToFloat64E(data)
		case schema.BooleanType:
			return cast.ToBoolE(data)
		case schema.ArrayType:
			slice := make([]any, 0)
			if err := json.Unmarshal([]byte(data), &slice); err != nil {
				return nil, err
			}
			return slice, nil
		case schema.ObjectType:
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
