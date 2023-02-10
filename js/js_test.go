package js

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	goroutineNum := 15
	blockNum := 4
	SetScheduler(NewScheduler(Options{InitialVMs: 2, MaxVMs: 4}))
	wg := new(sync.WaitGroup)

	for i := 1; i <= goroutineNum; i++ {
		wg.Add(1)
		go func(i int) {
			timeout := time.Second
			script := "1"
			if i < blockNum {
				script = `while(true){}`
				timeout = timeout * 2
			}

			ctx, _ := context.WithTimeout(context.Background(), timeout)
			defer func() {
				wg.Done()
			}()

			_, err := RunString(ctx, script)
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("%v: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	s := GetScheduler().(*schedulerImpl)
	initVMs := len(s.vms)
	if initVMs != s.initVMs {
		t.Fatalf("clean idle VM failed, want %v, got %v", s.initVMs, initVMs)
	}

	finalUnInitVMs := s.unInitVMs.Load()
	unInitVMs := int64(s.maxVMs - s.initVMs)
	if finalUnInitVMs != unInitVMs {
		t.Fatalf("clean idle VM failed, want %v, got %v", unInitVMs, finalUnInitVMs)
	}
}
