// Package v1 the version 1 api
package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
)

// msg the message struct
type msg struct {
	Msg string `json:"msg"`
}

// handleFunc handle error
type handleFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP handler responds to an HTTP request.
func (h handleFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		slog.Error("request error", "path", r.URL.Path)
		if e := JSON(w, http.StatusInternalServerError, msg{err.Error()}); e != nil {
			slog.Error("write response error", "error", e)
		}
	}
}

// JSON write JSON response.
func JSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set(contentType, mimeApplicationJSONCharsetUTF8)
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return err
	}
	return nil
}

// Routes the API server handler.
func Routes(secret string, timeout time.Duration, requestLog bool) http.Handler {
	root := chi.NewRouter()
	if requestLog {
		root.Use(requestLogger)
	}
	root.Use(auth(secret), middleware.Timeout(timeout))
	root.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if err := JSON(w, http.StatusNotFound, msg{http.StatusText(http.StatusNotFound)}); err != nil {
			slog.Error("write response error", "error", err)
		}
	})
	root.HandleFunc("/ping", ping)
	root.HandleFunc("/", ping)
	root.Route("/v1", func(api chi.Router) {
		api.Method(http.MethodPost, "/run", modelRun())
		api.Method(http.MethodPost, "/debug", modelDebug())
	})
	return root
}

func ping(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
