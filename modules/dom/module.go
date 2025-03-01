package dom

import (
	"sync/atomic"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/modules"
)

type toValue interface {
	toValue(this sobek.Value, rt *sobek.Runtime) sobek.Value
}

func init() {
	modules.Register("dom", modules.Global{
		"Event":       new(event),
		"EventTarget": new(eventTarget),
	})
}

var ids atomic.Uint32

func newID() uint32 { return ids.Add(1) }
