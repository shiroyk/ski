package js

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventLoop(t *testing.T) {
	t.Parallel()
	loop := NewEventLoop()
	var i int
	f := func() { i++ }
	loop.Start(f)
	assert.Equal(t, 1, i)
	loop.Start(f)
	assert.Equal(t, 2, i)
}

func TestEventLoopEnqueue(t *testing.T) {
	t.Parallel()
	loop := NewEventLoop()
	sleep := time.Millisecond * 500
	var i int
	f := func() {
		i++
		r := loop.EnqueueJob()
		go func() {
			time.Sleep(sleep)
			r(func() { i++ })
		}()
	}
	start := time.Now()
	loop.Start(f)
	took := time.Since(start)
	assert.Equal(t, 2, i)
	assert.Less(t, sleep, took)
}

func TestEventLoopAllJobCalled(t *testing.T) {
	t.Parallel()
	sleepTime := time.Millisecond * 500
	loop := NewEventLoop()
	var called int64
	f := func() {
		for i := 0; i < 10; i++ {
			bad := i == 9
			e := loop.EnqueueJob()

			go func() {
				if !bad {
					time.Sleep(sleepTime)
				}
				e(func() { atomic.AddInt64(&called, 1) })
			}()
		}
	}
	all := time.Now()
	for i := 0; i < 3; i++ {
		called = 0
		start := time.Now()
		loop.Start(f)
		took := time.Since(start)
		took2 := time.Since(start)
		assert.Less(t, time.Millisecond*500, took)
		assert.Less(t, sleepTime, took2)
		assert.Greater(t, sleepTime+time.Millisecond*100, took2)
		assert.EqualValues(t, 10, called)
	}
	took := time.Since(all)
	assert.Less(t, time.Millisecond*500*3, took)
}

func TestEventLoopPanicOnDoubleEnqueue(t *testing.T) {
	t.Parallel()
	loop := NewEventLoop()
	var i int
	f := func() {
		i++
		e := loop.EnqueueJob()
		go func() {
			time.Sleep(time.Second)
			e(func() { i++ })

			assert.Panics(t, func() { e(func() {}) })
		}()
	}
	start := time.Now()
	loop.Start(f)
	took := time.Since(start)
	assert.Equal(t, 2, i)
	assert.Less(t, time.Second, took)
	assert.Greater(t, time.Second+time.Millisecond*100, took)
}

func TestEventLoopStop(t *testing.T) {
	t.Parallel()
	loop := NewEventLoop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	go func() {
		<-ctx.Done()
		loop.Stop()
	}()

	start := time.Now()
	loop.Start(func() { loop.EnqueueJob() })
	<-ctx.Done()

	took := time.Since(start)
	assert.Less(t, time.Millisecond*500, took)
}

func TestEventLoopOnDone(t *testing.T) {
	t.Parallel()
	loop := NewEventLoop()
	var i int
	loop.Start(func() { loop.OnDone(func() { i++ }) })
	assert.Equal(t, 1, i)
}
