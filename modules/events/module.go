package events

import (
	"github.com/shiroyk/ski/modules"
)

func init() {
	modules.Register("events", modules.Global{
		"CustomEvent": new(customEvent),
		"Event":       new(_event),
		"EventTarget": new(_eventTarget),
	})
}
