package js

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/shiroyk/cloudcat"
)

func TestScheduler(t *testing.T) {
	goroutineNum := 20
	blockNum := 4
	scheduler := NewScheduler(Options{InitialVMs: 2, MaxVMs: 4})
	cloudcat.Provide(scheduler)
	wg := new(sync.WaitGroup)

	for i := 1; i <= goroutineNum; i++ {
		wg.Add(1)
		go func(i int) {
			timeout := time.Second
			script := "1"
			if i < blockNum {
				script = `while(true){}`
				timeout *= 2
			}

			ctx, _ := context.WithTimeout(context.Background(), timeout)
			defer func() {
				wg.Done()
			}()

			vm, err := scheduler.Get()
			if err != nil {
				t.Errorf("%v: %v", i, err)
				return
			}
			_, err = vm.RunString(ctx, script)
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("%v: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
}

func BenchmarkScheduler(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	wg := sync.WaitGroup{}
	for n := 0; n < b.N; n++ {
		wg.Add(1)
		go func() {
			_, _ = RunString(context.Background(), `1`)
			wg.Done()
		}()
	}
	b.StopTimer()
}
