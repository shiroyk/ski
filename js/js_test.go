package js

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/exp/rand"
)

func TestScheduler(t *testing.T) {
	concurrentNum := 15
	SetScheduler(NewScheduler(Options{InitialVMs: 2, MaxVMs: 4, MaxRetriesGetVM: 4}))
	wg := new(sync.WaitGroup)
	num := new(atomic.Int64)

	for i := 0; i < concurrentNum; i++ {
		wg.Add(1)
		go func(i int) {
			rand.Seed(uint64(i))
			timeout := time.Duration(rand.Intn(2)+1) * time.Second

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer func() {
				cancel()
				wg.Done()
			}()
			_, err := RunString(ctx, `while(true){}`)
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("%v: %v", i, err)
			} else {
				num.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if num.Load() != int64(concurrentNum) {
		t.Fatalf("concurrently run failed size %v", int64(concurrentNum)-num.Load())
	}

	s := defaultScheduler.Load().(*schedulerImpl)
	initial := len(s.vms)
	if s.activeVMs != initial || s.activeVMs != s.initVMs {
		t.Fatalf("active VMs %v should be equal to initial VMs %v", s.activeVMs, initial)
	}
}
