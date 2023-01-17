package parser

import (
	"sync"

	"golang.org/x/exp/slog"
)

var (
	manager = new(sync.Map)
)

type Parser interface {
	GetDesc() Desc
	GetString(*Context, any, string) (string, error)
	GetStrings(*Context, any, string) ([]string, error)
	GetElement(*Context, any, string) (string, error)
	GetElements(*Context, any, string) ([]string, error)
}

type Desc struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	ShortDesc string `json:"shortDesc"`
	LongDesc  string `json:"longDesc"`
	Url       string `json:"url"`
}

func Register(key string, parser Parser) {
	if _, ok := manager.Load(key); !ok {
		manager.Store(key, parser)
	} else {
		slog.Warn("parser %s already registered", key)
	}
}

func GetParser(key string) (Parser, bool) {
	if parser, ok := manager.Load(key); ok {
		return parser.(Parser), true
	}
	return nil, false
}

func GetAllDesc() []Desc {
	descList := make([]Desc, 0)
	Each(func(_ string, parser Parser) {
		descList = append(descList, parser.GetDesc())
	})
	return descList
}

func Each(f func(string, Parser)) {
	manager.Range(func(key, value any) bool {
		f(key.(string), value.(Parser))
		return true
	})
}
