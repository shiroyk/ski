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
	_ = rt.Set("setTimeout", t.setTimeout)
	_ = rt.Set("clearTimeout", t.clearTimeout)
	_ = rt.Set("setInterval", t.setInterval)
	_ = rt.Set("clearInterval", t.clearInterval)
	return nil, nil
}

func (*Timers) Global() {}

func (*Timers) setTimeout(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	callback, ok := sobek.AssertFunction(call.Argument(0))
	if !ok {
		panic(rt.NewTypeError("setTimeout: first argument must be a function"))
	}

	delay := time.Duration(call.Argument(1).ToInteger()) * time.Millisecond
	if delay < 0 {
		delay = 0
	}

	var args []sobek.Value
	if len(call.Arguments) > 2 {
		args = call.Arguments[2:]
	}

	ctx := js.Context(rt)
	enqueue := js.EnqueueJob(rt)
	t := rtTimers(rt).new(delay, false)
	task := func() error {
		t.stop()
		_, err := callback(sobek.Undefined(), args...)
		return err
	}

	go func() {
		select {
		case <-t.timer.C:
			enqueue(task)
		case <-t.done:
			enqueue(func() error { return nil })
		case <-ctx.Done():
			t.stop()
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

	delay := time.Duration(call.Argument(1).ToInteger()) * time.Millisecond
	if delay < 0 {
		delay = 0
	}

	var args []sobek.Value
	if len(call.Arguments) > 2 {
		args = call.Arguments[2:]
	}

	ctx := js.Context(rt)
	enqueue := js.EnqueueJob(rt)
	t := rtTimers(rt).new(delay, true)
	task := func() error { _, err := callback(sobek.Undefined(), args...); return err }

	go func() {
		for {
			select {
			case <-t.ticker.C:
				enqueue(task)
				enqueue = js.EnqueueJob(rt)
			case <-t.done:
				enqueue(func() error { return nil })
				return
			case <-ctx.Done():
				t.stop()
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
	timer   *time.Timer
	ticker  *time.Ticker
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
	if t.timer != nil {
		t.timer.Stop()
	}
	if t.ticker != nil {
		t.ticker.Stop()
	}
	t.cleanup()
}

type timers struct {
	id    int64
	timer map[int64]*timer
}

func (t *timers) new(delay time.Duration, repeat bool) *timer {
	t.id++
	id := t.id
	nt := &timer{
		id:      id,
		done:    make(chan struct{}),
		cleanup: func() { delete(t.timer, id) },
	}
	if repeat {
		nt.ticker = time.NewTicker(delay)
	} else {
		nt.timer = time.NewTimer(delay)
	}
	t.timer[id] = nt
	return nt
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
