package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/promise"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules"
	"github.com/shiroyk/ski/modules/fetch"
	"github.com/shiroyk/ski/modules/signal"
)

func init() {
	modules.Register("http/server", new(Server))
}

// Server an HTTP server implementation.
type Server struct{}

func (s *Server) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := s.prototype(rt)
	ctor := rt.ToValue(s.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}

func (s *Server) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("listening", rt.ToValue(s.listening), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("addr", rt.ToValue(s.addr), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("url", rt.ToValue(s.url), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("finished", rt.ToValue(s.finished), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)

	_ = p.Set("ref", s.ref)
	_ = p.Set("unref", s.unref)
	_ = p.Set("shutdown", s.shutdown)
	_ = p.Set("close", s.close)
	return p
}

func (s *Server) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	serv := &httpServer{
		rt:       rt,
		port:     8000,
		hostname: "127.0.0.1",
		ctx:      context.Background(),
		server:   &http.Server{Addr: "127.0.0.1:8000"},
	}

	if len(call.Arguments) == 0 {
		panic(rt.NewTypeError("serve requires at least one argument"))
	}

	var handler sobek.Value

	opt := call.Argument(0)
	switch {
	case types.IsNumber(opt):
		port := opt.ToInteger()
		if port <= 0 {
			panic(rt.NewTypeError("port must be a positive number"))
		}
		serv.port = int(port)
		serv.server.Addr = fmt.Sprintf(":%d", serv.port)
		handler = call.Argument(1)
	case types.IsFunc(opt):
		handler = opt
	default:
		opts := opt.ToObject(rt)
		if v := opts.Get("port"); v != nil {
			serv.port = int(v.ToInteger())
			serv.server.Addr = fmt.Sprintf(":%d", serv.port)
		}
		if v := opts.Get("hostname"); v != nil {
			serv.hostname = v.String()
			serv.server.Addr = fmt.Sprintf("%s:%d", serv.hostname, serv.port)
		}
		if v := opts.Get("maxHeaderSize"); v != nil {
			serv.server.MaxHeaderBytes = int(v.ToInteger())
		}
		if v := opts.Get("keepAliveTimeout"); v != nil {
			serv.server.IdleTimeout = time.Duration(v.ToInteger()) * time.Millisecond
		}
		if v := opts.Get("requestTimeout"); v != nil {
			serv.server.ReadTimeout = time.Duration(v.ToInteger()) * time.Millisecond
		}
		if v := opts.Get("signal"); v != nil {
			if v.ExportType() != signal.TypeAbortSignal {
				panic(rt.NewTypeError("signal must be an AbortSignal"))
			}
			serv.ctx = signal.Context(rt, v)
			context.AfterFunc(serv.ctx, func() { serv.shutdown() })
		}
		if v := opts.Get("onError"); v != nil {
			var ok bool
			serv.onError, ok = sobek.AssertFunction(v)
			if !ok {
				panic(rt.NewTypeError("onError must be a function"))
			}
		}
		if v := opts.Get("onListen"); v != nil {
			var ok bool
			serv.onListen, ok = sobek.AssertFunction(v)
			if !ok {
				panic(rt.NewTypeError("onListen must be a function"))
			}
		}
		if v := opts.Get("handler"); v != nil {
			handler = v
		}
		if v := call.Argument(1); !sobek.IsUndefined(v) {
			handler = v
		}
	}

	if handler != nil {
		var ok bool
		serv.handler, ok = sobek.AssertFunction(handler)
		if !ok {
			panic(rt.NewTypeError("handler must be a function"))
		}
	}
	if serv.onError == nil {
		serv.onError = func(this sobek.Value, args ...sobek.Value) (sobek.Value, error) {
			code := http.StatusInternalServerError
			msg := http.StatusText(code)
			if len(args) > 0 {
				err := args[0].ToObject(rt)
				url := err.Get("url").String()
				method := err.Get("method").String()
				message := err.Get("message").String()
				msg = fmt.Sprintf("Internal Server Error %s %s %s", method, url, message)
			}
			slog.Error(msg, slog.String("source", "server"))
			return fetch.NewResponse(rt, &http.Response{
				StatusCode: code,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(http.StatusText(code))),
			}), nil
		}
	}
	if serv.handler == nil {
		serv.handler = func(this sobek.Value, args ...sobek.Value) (sobek.Value, error) {
			code := http.StatusNotFound
			body := strings.NewReader(http.StatusText(http.StatusNotFound))
			return fetch.NewResponse(rt, &http.Response{
				StatusCode: code,
				Header:     make(http.Header),
				Body:       io.NopCloser(body),
			}), nil
		}
	}
	if serv.server.Addr == "" {
		serv.port = 8000
		serv.hostname = "127.0.0.1"
		serv.server.Addr = "127.0.0.1:8000"
	}
	serv.server.Handler = serv
	serv.ref = js.EnqueueJob(rt)
	ln := serv.listen()

	go func() {
		js.EnqueueJob(rt)(func() error {
			if serv.onListen != nil {
				_, _ = serv.onListen(sobek.Undefined(), serv.addr())
			} else {
				slog.Info(fmt.Sprintf("listening on %s", serv.url()),
					slog.String("source", "server"))
			}
			return nil
		})
		err := serv.server.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			js.EnqueueJob(rt)(func() error { return err })
		}
	}()

	obj := rt.NewObject()
	_ = obj.SetSymbol(symHttpServer, rt.ToValue(serv))
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (s *Server) listening(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHttpServer(rt, call.This)
	return rt.ToValue(!this.closed.Load())
}

func (s *Server) addr(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHttpServer(rt, call.This)
	return this.addr()
}

func (s *Server) url(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHttpServer(rt, call.This)
	return rt.ToValue(this.url())
}

func (s *Server) finished(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHttpServer(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		<-this.ctx.Done()
		callback(func() (any, error) { return nil, nil })
	})
}

func (s *Server) ref(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHttpServer(rt, call.This)
	if this.ref == nil {
		this.ref = js.EnqueueJob(rt)
	}
	return call.This
}

func (s *Server) unref(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHttpServer(rt, call.This)
	this.ref(func() error { this.ref = nil; return nil })
	return call.This
}

func (s *Server) close(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHttpServer(rt, call.This)
	if err := this.close(); err != nil {
		js.Throw(rt, err)
	}
	return sobek.Undefined()
}

func (s *Server) shutdown(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toHttpServer(rt, call.This)
	return promise.New(rt, func(callback promise.Callback) {
		err := this.shutdown()
		callback(func() (any, error) { return nil, err })
	})
}

type httpServer struct {
	rt       *sobek.Runtime
	server   *http.Server
	hostname string
	port     int

	handler, onError, onListen sobek.Callable

	ctx    context.Context
	closed atomic.Bool

	ref js.Enqueue
}

func (s *httpServer) url() string {
	if s.port == 80 {
		return "http://" + s.hostname
	}
	return fmt.Sprintf("http://%s:%d", s.hostname, s.port)
}

func (s *httpServer) addr() sobek.Value {
	return s.rt.ToValue(map[string]any{
		"hostname": s.hostname,
		"port":     s.port,
	})
}

func (s *httpServer) listen() net.Listener {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		js.Throw(s.rt, err)
	}
	return ln
}

func (s *httpServer) close() error {
	s.closed.Store(true)
	err := s.server.Close()
	if s.ref != nil {
		s.ref(func() error { s.ref = nil; return nil })
	}
	return err
}

func (s *httpServer) shutdown() error {
	s.closed.Store(true)
	err := s.server.Shutdown(s.ctx)
	if s.ref != nil {
		s.ref(func() error { s.ref = nil; return nil })
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// ServeHTTP implements http.Handler
func (s *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	wg.Add(1)
	js.EnqueueJob(s.rt)(func() error {
		result, err := s.handler(sobek.Undefined(), fetch.NewRequest(s.rt, r))
		if err != nil {
			s.writeError(w, r, wg.Done, err)
			return nil
		}

		// Handle promise result
		if types.IsPromise(result) {
			s.handlePromise(w, r, wg.Done, result)
			return nil
		}

		if res, ok := fetch.ToResponse(result); ok {
			s.writeResponse(w, r, wg.Done, res)
		} else {
			s.writeError(w, r, wg.Done, errNotResponse)
		}
		return nil
	})
	wg.Wait()
}

func (s *httpServer) writeResponse(w http.ResponseWriter, r *http.Request, done func(), res *http.Response) {
	defer done()

	header := w.Header()
	for k, v := range res.Header {
		header[http.CanonicalHeaderKey(k)] = v
	}
	w.WriteHeader(res.StatusCode)

	if _, err := io.Copy(w, res.Body); err != nil {
		slog.Error("Failed to write response",
			slog.String("source", "server"),
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()))
	}
}

func (s *httpServer) writeError(w http.ResponseWriter, r *http.Request, done func(), rawErr error) {
	var (
		jsErr  *sobek.Object
		result sobek.Value
		err    error
	)

	jsErr, err = s.rt.New(s.rt.Get("Error"), s.rt.ToValue(rawErr.Error()))
	if err != nil {
		goto EX
	}

	_ = jsErr.Set("method", r.Method)
	_ = jsErr.Set("url", r.URL.String())

	result, err = s.onError(sobek.Undefined(), jsErr)
	if err != nil {
		goto EX
	}

	if !types.IsPromise(result) {
		if res, ok := fetch.ToResponse(result); ok {
			s.writeResponse(w, r, done, res)
			return
		}
		err = errNotResponse
	} else {
		switch p := result.Export().(*sobek.Promise); p.State() {
		case sobek.PromiseStateRejected:
			if ex, ok := p.Result().Export().(error); ok {
				err = ex
			} else {
				err = errors.New(p.Result().String())
			}
		case sobek.PromiseStateFulfilled:
			if res, ok := fetch.ToResponse(result); ok {
				s.writeResponse(w, r, done, res)
				return
			}
			err = errNotResponse
		default:
			if err = s.handlePendingPromise(w, r, done, result); err == nil {
				return
			}
		}
	}

EX:
	slog.Error("Exception in onError while handling exception", slog.String("message", err.Error()))
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(internalServerError)
	done()
}

// handlePromise handles promise result
func (s *httpServer) handlePromise(w http.ResponseWriter, r *http.Request, done func(), result sobek.Value) {
	var err error
	switch p := result.Export().(*sobek.Promise); p.State() {
	case sobek.PromiseStateRejected:
		if ex, ok := p.Result().Export().(error); ok {
			err = ex
		} else {
			err = errors.New(p.Result().String())
		}
	case sobek.PromiseStateFulfilled:
		if res, ok := fetch.ToResponse(p.Result()); ok {
			s.writeResponse(w, r, done, res)
		} else {
			err = errNotResponse
		}
	default:
		err = s.handlePendingPromise(w, r, done, result)
	}
	if err != nil {
		s.writeError(w, r, done, err)
	}
}

// handlePendingPromise handles a pending promise with resolve and reject callbacks
func (s *httpServer) handlePendingPromise(w http.ResponseWriter, r *http.Request, done func(), promise sobek.Value) error {
	object := promise.(*sobek.Object)
	then, ok := sobek.AssertFunction(object.Get("then"))
	if !ok {
		return errNotResponse
	}

	resolve := s.rt.ToValue(func(call sobek.FunctionCall) sobek.Value {
		if res, ok := fetch.ToResponse(call.Argument(0)); ok {
			s.writeResponse(w, r, done, res)
		} else {
			s.writeError(w, r, done, errNotResponse)
		}
		return sobek.Undefined()
	})

	reject := s.rt.ToValue(func(call sobek.FunctionCall) sobek.Value {
		v := call.Argument(0)
		if v.ExportType() == types.TypeError {
			s.writeError(w, r, done, v.Export().(error))
		} else {
			s.writeError(w, r, done, errors.New(v.String()))
		}
		return sobek.Undefined()
	})

	if _, err := then(object, resolve, reject); err != nil {
		return err
	}
	return nil
}

var (
	internalServerError = []byte(http.StatusText(http.StatusInternalServerError))
	errNotResponse      = errors.New("return value from handler must be a response or a promise resolving to a response")
	symHttpServer       = sobek.NewSymbol("Symbol.HttpServer")
)

func toHttpServer(rt *sobek.Runtime, value sobek.Value) *httpServer {
	if o, ok := value.(*sobek.Object); ok {
		if v := o.GetSymbol(symHttpServer); v != nil {
			return v.Export().(*httpServer)
		}
	}
	panic(rt.NewTypeError(`Value of "this" must be of type HttpServer`))
}
