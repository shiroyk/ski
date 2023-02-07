package middleware

import (
	"github.com/shiroyk/cloudcat/fetch"
	"github.com/shiroyk/cloudcat/js"
	"golang.org/x/exp/slog"
)

type Rewrite struct{}

func (re *Rewrite) ProcessRequest(r *fetch.Request) {
	rewrite := r.Context().Value("rewrite")
	if script, ok := rewrite.(string); ok {
		_, err := js.Run(r.Context(), js.Program{Code: script, Args: map[string]any{"request": r}})
		if err != nil {
			slog.FromContext(r.Context()).Error("rewrite error:", err)
		}
	}
}

func (re *Rewrite) ProcessResponse(r *fetch.Response) {
	if r.Request == nil {
		return
	}
	ctx := r.Request.Context()
	rewrite := ctx.Value("rewrite")
	if script, ok := rewrite.(string); ok {
		_, err := js.Run(ctx, js.Program{Code: script, Args: map[string]any{"response": r}})
		if err != nil {
			slog.FromContext(ctx).Error("rewrite error:", err)
		}
	}
}
