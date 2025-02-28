package modulestest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/promise"
)

// HttpServer creates an HTTP server for testing
func HttpServer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	var addr string
	var handler sobek.Callable
	arg := call.Argument(0)
	if !sobek.IsUndefined(arg) {
		switch v := arg.Export().(type) {
		case string:
			addr = v
			arg = call.Argument(1)
		case int64:
			addr = fmt.Sprintf("127.0.0.1:%d", v)
			arg = call.Argument(1)
		}
	}
	var ok bool
	handler, ok = sobek.AssertFunction(arg)
	if !ok {
		panic(rt.NewTypeError("handler must be a function"))
	}
	if addr == "" {
		addr = "127.0.0.1:0"
	}

	listen, err := net.Listen("tcp", addr)
	if err != nil {
		js.Throw(rt, err)
	}
	addr = listen.Addr().String()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			done := make(chan struct{})
			js.EnqueueJob(rt)(func() error {
				req := &request{
					req:      r,
					Method:   r.Method,
					Host:     r.Host,
					Path:     r.URL.Path,
					Protocol: r.Proto,
					Body:     &body{b: r.Body},
				}
				res := &response{
					res:  w,
					done: done,
				}
				v, err := handler(sobek.Undefined(), rt.ToValue(req), rt.ToValue(res))
				if err == nil {
					_, err = promise.Result(v)
				}
				if err == nil {
					return nil
				}
				slog.Error("test server: error", "method", r.Method, "path", r.URL.Path, "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, err.Error())
				close(done)
				return nil
			})
			<-done
		}),
	}

	enqueue := js.EnqueueJob(rt)
	go func() {
		slog.Info("test server: http://" + addr)
		err = server.Serve(listen)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		enqueue(func() error { return err })
	}()

	js.Cleanup(rt, func() { server.Close() })

	ret := rt.NewObject()
	_ = ret.Set("url", "http://"+addr)
	_ = ret.Set("close", func() { server.Close() })

	return ret
}

type request struct {
	req      *http.Request
	Path     string
	Method   string
	Host     string
	Protocol string
	Body     *body
}

type body struct {
	b    io.ReadCloser
	used bool
}

func (r *body) read(rt *sobek.Runtime) []byte {
	if r.used {
		js.Throw(rt, errors.New("body already used"))
	}
	r.used = true
	defer r.b.Close()
	data, err := io.ReadAll(r.b)
	if err != nil {
		js.Throw(rt, err)
	}
	return data
}

func (r *body) Json(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	var v any
	err := json.Unmarshal(r.read(rt), &v)
	if err != nil {
		js.Throw(rt, err)
	}
	return rt.ToValue(v)
}

func (r *body) Text(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(r.read(rt))
}

func (r *body) Arraybuffer(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(r.read(rt))
}

func (r *request) GetHeaders(_ sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(r.req.Header)
}

func (r *request) GetHeader(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(r.req.Header.Get(call.Argument(0).String()))
}

type response struct {
	res        http.ResponseWriter
	done       chan struct{}
	StatusCode int
}

func (r *response) RemoveHeader(call sobek.FunctionCall) sobek.Value {
	r.res.Header().Del(call.Argument(0).String())
	return sobek.Undefined()
}

func (r *response) GetHeader(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(r.res.Header().Get(call.Argument(0).String()))
}

func (r *response) SetHeader(call sobek.FunctionCall) sobek.Value {
	r.res.Header().Set(call.Argument(0).String(), call.Argument(1).String())
	return sobek.Undefined()
}

func (r *response) Write(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	data, err := js.Unwrap(call.Argument(0))
	if err != nil {
		r.res.WriteHeader(http.StatusInternalServerError)
		r.res.Write([]byte(err.Error()))
		close(r.done)
		js.Throw(rt, err)
	}

	r.res.Write([]byte(fmt.Sprintf("%v", data)))

	return sobek.Undefined()
}

func (r *response) End(call sobek.FunctionCall) sobek.Value {
	data, err := js.Unwrap(call.Argument(0))
	if err != nil {
		r.res.WriteHeader(http.StatusInternalServerError)
		r.res.Write([]byte(err.Error()))
		goto END
	}

	if r.StatusCode != 0 {
		r.res.WriteHeader(r.StatusCode)
	}
	r.res.Write([]byte(fmt.Sprintf("%v", data)))

END:
	close(r.done)
	return sobek.Undefined()
}
