package v1

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/shiroyk/cloudcat/lib/logger"
)

// Msg the message struct
type Msg struct {
	Msg string `json:"msg"`
}

// HandleFunc handle error
type HandleFunc func(w *Response, r *http.Request) error

// ServeHTTP handler responds to an HTTP request.
func (h HandleFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := &Response{w, 200, false}
	if err := h(res, r); err != nil {
		logger.Error("request error:", err, "path", r.URL.Path)
		if e := res.JSON(http.StatusInternalServerError, Msg{err.Error()}); e != nil {
			logger.Error("write response error:", e)
		}
	}
	logger.Debug("request", "path", r.URL.Path, "method", r.Method, "status", res.status)
}

// Response the http.ResponseWriter wrapper
type Response struct {
	http.ResponseWriter
	status    int
	committed bool
}

// JSON write JSON response
func (r *Response) JSON(status int, data any) error {
	if r.committed {
		return nil
	}
	if !strings.Contains(r.Header().Get("Content-Type"), "json") {
		r.Header().Set(contentType, mimeApplicationJSONCharsetUTF8)
	}
	if status != http.StatusOK {
		r.WriteHeader(status)
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = r.Write(bytes)
	if err != nil {
		return err
	}

	r.Flush()

	return nil
}

// Flush sends any buffered data to the client.
func (r *Response) Flush() {
	r.ResponseWriter.(http.Flusher).Flush()
}

// WriteHeader sends an HTTP response header with the provided
// status code.
func (r *Response) WriteHeader(code int) {
	r.committed = true
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// auth authentication the token
func auth(token string, next http.Handler) http.Handler {
	return HandleFunc(func(w *Response, r *http.Request) error {
		if token != strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") {
			err := w.JSON(http.StatusUnauthorized, Msg{http.StatusText(http.StatusUnauthorized)})
			return err
		}

		next.ServeHTTP(w, r)

		return nil
	})
}
