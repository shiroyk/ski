package js

import (
	"sync"
)

// EventLoop implements an eventloop.
type EventLoop struct {
	queue    []func()   // queue to store the job to be executed
	doneJobs []func()   // job of Done
	enqueue  uint       // Count of job in the event loop
	cond     *sync.Cond // Condition variable for synchronization
}

// NewEventLoop create a new EventLoop instance
func NewEventLoop() *EventLoop {
	return &EventLoop{
		cond:     sync.NewCond(new(sync.Mutex)),
		doneJobs: make([]func(), 0),
	}
}

// Start the event loop and execute the provided function
func (e *EventLoop) Start(f func()) {
	e.cond.L.Lock()
	e.queue = []func(){f}
	e.cond.L.Unlock()
	for {
		e.cond.L.Lock()

		if len(e.queue) > 0 {
			queue := e.queue
			e.queue = make([]func(), 0, len(queue))
			e.cond.L.Unlock()

			for _, job := range queue {
				job()
			}
			continue
		}

		if e.enqueue > 0 {
			e.cond.Wait()
			e.cond.L.Unlock()
			continue
		}

		done := e.doneJobs
		e.doneJobs = e.doneJobs[:0]
		e.cond.L.Unlock()

		if len(done) > 0 {
			for _, job := range done {
				job()
			}
		}

		return
	}
}

type Enqueue func(func())

// EnqueueJob return a function Enqueue to add a job to the job queue.
// Usage:
//
//	func main() {
//		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			w.WriteHeader(http.StatusOK)
//			_, _ = w.Write([]byte(`{"foo":"bar"}`))
//		}))
//		defer server.Close()
//
//		loop := NewEventLoop()
//		runtime := sobek.New()
//
//		_ = runtime.Set("fetch", func(call sobek.FunctionCall) sobek.Value {
//			promise, resolve, reject := runtime.NewPromise()
//			enqueue := loop.EnqueueJob()
//
//			go func() {
//				res, err := http.Get(call.Argument(0).String())
//				if err != nil {
//					enqueue(func() { reject(err) })
//					return
//				}
//				loop.OnDone(func() { res.Body.Close() })
//
//				data, err := io.ReadAll(res.Body)
//				if err != nil {
//					enqueue(func() { reject(err) })
//					return
//				}
//
//				enqueue(func() { resolve(string(data)) })
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
//		loop.Start(func() { ret, err = runtime.RunString(fmt.Sprintf(`fetch("%s")`, server.URL)) })
//
//		if err != nil {
//			fmt.Println(err)
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
//			fmt.Println(promise.Result().String())
//		case sobek.PromiseStateFulfilled:
//			fmt.Println(promise.Result().Export())
//		}
//	}
func (e *EventLoop) EnqueueJob() Enqueue {
	e.cond.L.Lock()
	called := false
	e.enqueue++
	e.cond.L.Unlock()
	return func(job func()) {
		e.cond.L.Lock()
		if called {
			e.cond.L.Unlock()
			panic("Enqueue already called")
		}
		e.queue = append(e.queue, job) // Add the job to the queue
		called = true
		e.enqueue--
		e.cond.Signal() // Signal the condition variable
		e.cond.L.Unlock()
	}
}

// Stop the eventloop
func (e *EventLoop) Stop() {
	e.cond.L.Lock()
	defer e.cond.L.Unlock()
	// clean the queue
	e.queue = e.queue[:0]
	e.enqueue = 0
	e.doneJobs = e.doneJobs[:0]
	e.cond.Signal()
}

// OnDone add a function to execute when done.
func (e *EventLoop) OnDone(job func()) {
	e.cond.L.Lock()
	defer e.cond.L.Unlock()

	e.doneJobs = append(e.doneJobs, job)
}
