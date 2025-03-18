package js

import (
	"sync"

	"github.com/grafana/sobek"
)

// EventLoop implements an eventloop.
type EventLoop struct {
	queue   []func() error // queue to store the job to be executed
	cleanup []func()       // job of cleanup
	enqueue uint           // Count of job in the event loop
	cond    *sync.Cond     // Condition variable for synchronization
}

// NewEventLoop create a new EventLoop instance
func NewEventLoop() *EventLoop {
	return &EventLoop{
		cond:    sync.NewCond(new(sync.Mutex)),
		cleanup: make([]func(), 0),
	}
}

// Start the event loop and execute the provided function
func (e *EventLoop) Start(task func() error) (err error) {
	e.cond.L.Lock()
	e.queue = []func() error{task}
	e.cond.L.Unlock()
	for {
		e.cond.L.Lock()

		if len(e.queue) > 0 {
			queue := e.queue
			e.queue = make([]func() error, 0, len(queue))
			e.cond.L.Unlock()

			for _, job := range queue {
				if err2 := job(); err2 != nil {
					if err != nil {
						err = append(err.(joinError), err2)
					} else {
						err = joinError{err2}
					}
				}
			}
			continue
		}

		if e.enqueue > 0 {
			e.cond.Wait()
			e.cond.L.Unlock()
			continue
		}

		if len(e.cleanup) > 0 {
			cleanup := e.cleanup
			e.cleanup = e.cleanup[:0]
			e.cond.L.Unlock()

			for _, clean := range cleanup {
				clean()
			}
		} else {
			e.cond.L.Unlock()
		}

		return
	}
}

// Enqueue add a job to the job queue.
type Enqueue func(func() error)

// EnqueueJob return a function Enqueue to add a job to the job queue.
// Usage:
//
//		func main() {
//		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			w.WriteHeader(http.StatusOK)
//			_, _ = w.Write([]byte(`{"foo":"bar"}`))
//		}))
//		defer server.Close()
//
//		loop := js.NewEventLoop()
//		runtime := sobek.New()
//
//		_ = runtime.Set("fetch", func(call sobek.FunctionCall) sobek.Value {
//			promise, resolve, reject := runtime.NewPromise()
//			enqueue := loop.EnqueueJob()
//
//			go func() {
//				res, err := http.Get(call.Argument(0).String())
//				if err != nil {
//					enqueue(func() error { return reject(err) })
//					return
//				}
//				loop.Cleanup(func() { res.Body.Close() })
//
//				data, err := io.ReadAll(res.Body)
//				if err != nil {
//					enqueue(func() error { return reject(err) })
//					return
//				}
//
//				enqueue(func() error { return resolve(string(data)) })
//			}()
//
//			return runtime.ToValue(promise)
//		})
//
//		var (
//			ret sobek.Value
//			err error
//		)
//
//		err = loop.Start(func() error {
//			ret, err = runtime.RunString(fmt.Sprintf(`fetch("%s")`, server.URL))
//			return err
//		})
//
//		if err != nil {
//			panic(err)
//		}
//		promise, ok := ret.Export().(*sobek.Promise)
//		if !ok {
//			panic("expect promise")
//			return
//		}
//
//		switch promise.State() {
//		case sobek.PromiseStatePending:
//			panic("unexpect pending state")
//		case sobek.PromiseStateRejected:
//			panic(promise.Result().(error))
//		case sobek.PromiseStateFulfilled:
//			fmt.Println(promise.Result().Export())
//		}
//	}
func (e *EventLoop) EnqueueJob() Enqueue {
	e.cond.L.Lock()
	called := false
	e.enqueue++
	e.cond.L.Unlock()
	return func(job func() error) {
		e.cond.L.Lock()
		defer e.cond.L.Unlock()
		switch {
		case called:
			panic("Enqueue already called")
		case e.enqueue == 0:
			return // Eventloop stopped
		}
		e.queue = append(e.queue, job) // Add the job to the queue
		called = true
		e.enqueue--
		e.cond.Signal() // Signal the condition variable
	}
}

// Stop the eventloop with the provided error
func (e *EventLoop) Stop(err error) {
	e.cond.L.Lock()
	defer e.cond.L.Unlock()
	// clean the queue
	e.queue = append(e.queue[:0], func() error { return err })
	e.enqueue = 0
	e.cond.Signal()
}

// Cleanup add a function to execute when run finish.
func (e *EventLoop) Cleanup(job ...func()) {
	e.cond.L.Lock()
	defer e.cond.L.Unlock()

	e.cleanup = append(e.cleanup, job...)
}

// EnqueueJob return a function Enqueue to add a job to the job queue.
func EnqueueJob(rt *sobek.Runtime) Enqueue { return self(rt).eventloop.EnqueueJob() }

// Cleanup add a function to execute when the VM has finished running.
// eg: close resources...
func Cleanup(rt *sobek.Runtime, job ...func()) { self(rt).eventloop.Cleanup(job...) }
