package timers

import (
	"time"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/modules"
)

// Timers implements JavaScript timer functions
type Timers struct{}

func init() {
	modules.Register("timers", new(Timers))
}

func (t *Timers) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	_ = rt.GlobalObject().SetSymbol(symTimers, &timers{timer: make(map[int64]*timer)})
	ret := rt.NewObject()
	_ = ret.Set("setTimeout", t.setTimeout)
	_ = ret.Set("clearTimeout", t.clearTimeout)
	_ = ret.Set("setInterval", t.setInterval)
	_ = ret.Set("clearInterval", t.clearInterval)
	return ret, nil
}

func (t *Timers) Global() {}

func (*Timers) setTimeout(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	callback, ok := sobek.AssertFunction(call.Argument(0))
	if !ok {
		panic(rt.NewTypeError("setTimeout: first argument must be a function"))
	}

	i := call.Argument(1).ToInteger()
	if i < 1 || i > 2147483647 {
		i = 1
	}
	delay := time.Duration(i) * time.Millisecond

	var args []sobek.Value
	if len(call.Arguments) > 2 {
		args = call.Arguments[2:]
	}

	enqueue := js.EnqueueJob(rt)
	t := rtTimers(rt).new(delay, false)
	js.Cleanup(rt, t.stop)
	task := func() error {
		defer t.stop()
		_, err := callback(sobek.Undefined(), args...)
		return err
	}

	go func() {
		select {
		case <-t.timer:
			enqueue(task)
		case <-t.done:
			enqueue(nothing)
		}
	}()

	return rt.ToValue(t.id)
}

func (*Timers) clearTimeout(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	id := call.Argument(0).ToInteger()
	rtTimers(rt).stop(id)
	return sobek.Undefined()
}

func (*Timers) setInterval(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	callback, ok := sobek.AssertFunction(call.Argument(0))
	if !ok {
		panic(rt.NewTypeError("setInterval: first argument must be a function"))
	}

	i := call.Argument(1).ToInteger()
	if i < 1 || i > 2147483647 {
		i = 1
	}
	delay := time.Duration(i) * time.Millisecond

	var args []sobek.Value
	if len(call.Arguments) > 2 {
		args = call.Arguments[2:]
	}

	enqueue := js.EnqueueJob(rt)
	t := rtTimers(rt).new(delay, true)
	js.Cleanup(rt, t.stop)
	task := func() error { _, err := callback(sobek.Undefined(), args...); return err }

	go func() {
		for {
			select {
			case <-t.timer:
				enqueue(task)
				enqueue = js.EnqueueJob(rt)
			case <-t.done:
				enqueue(nothing)
				return
			}
		}
	}()

	return rt.ToValue(t.id)
}

func (*Timers) clearInterval(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	id := call.Argument(0).ToInteger()
	rtTimers(rt).stop(id)
	return sobek.Undefined()
}

type timer struct {
	id      int64
	timer   <-chan time.Time
	done    chan struct{}
	cleanup func()
}

func (t *timer) stop() {
	select {
	case _, ok := <-t.done:
		if !ok {
			return
		}
	default:
	}
	close(t.done)
	t.cleanup()
}

type timers struct {
	id    int64
	timer map[int64]*timer
}

func (t *timers) new(delay time.Duration, repeat bool) *timer {
	t.id++
	id := t.id
	n := &timer{
		id:   id,
		done: make(chan struct{}),
	}
	if repeat {
		t1 := time.NewTicker(delay)
		n.timer = t1.C
		n.cleanup = func() {
			delete(t.timer, id)
			t1.Stop()
		}
	} else {
		t1 := time.NewTimer(delay)
		n.timer = t1.C
		n.cleanup = func() {
			delete(t.timer, id)
			t1.Stop()
		}
	}
	t.timer[id] = n
	return n
}

func (t *timers) stop(id int64) {
	if v, ok := t.timer[id]; ok {
		v.stop()
	}
}

var symTimers = sobek.NewSymbol(`Symbol.__timers__`)

func rtTimers(rt *sobek.Runtime) *timers {
	t, ok := rt.GlobalObject().GetSymbol(symTimers).Export().(*timers)
	if ok {
		return t
	}
	panic(rt.NewTypeError(`symbol value of "timers" must be Timers`))
}

func nothing() error { return nil }
