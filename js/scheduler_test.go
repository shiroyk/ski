package js

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScheduler(t *testing.T) {
	scheduler := NewScheduler(SchedulerOptions{InitialVMs: 2, MaxVMs: 4})
	goroutineNum := 12
	blockNum := 4
	wg := new(sync.WaitGroup)

	for i := 1; i <= goroutineNum; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			timeout := time.Millisecond * 400
			script := "1"
			if i < blockNum {
				script = `while(true){}`
				timeout *= 2
			}

			vm, err := scheduler.Get()
			if err != nil {
				t.Errorf("scheduler %v: %v", i, err)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			vm.Run(ctx, func() {
				_, err := vm.Runtime().RunString(script)
				if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
					t.Errorf("run string %v: %v", i, err)
				}
			})
		}(i)
	}
	wg.Wait()
}

func TestSchedulerShrink(t *testing.T) {
	scheduler := NewScheduler(SchedulerOptions{InitialVMs: 2, MaxVMs: 4})
	scheduler.Shrink()
	assert.Equal(t, `{"available":0,"max":4,"unInit":4}`, scheduler.(fmt.Stringer).String())
	start := time.Now()
	_, _ = scheduler.Get()
	_, _ = scheduler.Get()
	took := time.Since(start)
	assert.Equal(t, `{"available":0,"max":4,"unInit":2}`, scheduler.(fmt.Stringer).String())
	assert.True(t, took < time.Millisecond*600)
}
