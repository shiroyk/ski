package js

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/logger"
	"github.com/shiroyk/cloudcat/utils"
)

const (
	DefaultMaxTimeToWaitGetVM = 500 * time.Millisecond
	DefaultMaxRetriesGetVM    = 3
)

var (
	defaultScheduler   atomic.Value
	ErrSchedulerClosed = errors.New("scheduler is closed")
)

func init() {
	defaultScheduler.Store(NewScheduler(Options{InitialVMs: 2, MaxVMs: runtime.GOMAXPROCS(0)}))
}

// SetScheduler makes s the default Scheduler.
func SetScheduler(s Scheduler) {
	defaultScheduler.Store(s)
}

// GetScheduler returns the default Scheduler.
func GetScheduler() Scheduler {
	return defaultScheduler.Load().(Scheduler)
}

// RunString the js string
func RunString(ctx context.Context, script string) (goja.Value, error) {
	tr, err := GetScheduler().Get()
	if err != nil {
		return nil, err
	}
	return tr.RunString(ctx, script)
}

// Run the js program
func Run(ctx context.Context, p common.Program) (goja.Value, error) {
	tr, err := GetScheduler().Get()
	if err != nil {
		return nil, err
	}
	return tr.Run(ctx, p)
}

// Scheduler the VM scheduler
type Scheduler interface {
	// Get the VM
	Get() (VM, error)
	// Release the VM
	Release(VM)
	// Close the scheduler
	Close() error
}

// Options Scheduler options
type Options struct {
	InitialVMs, MaxVMs, MaxRetriesGetVM int
	MaxTimeToWaitGetVM                  time.Duration
	UseStrict                           bool
}

type schedulerImpl struct {
	mu                               *sync.Mutex
	vms                              chan VM
	initVMs, maxVMs, maxRetriesGetVM int
	unInitVMs, waiting               *atomic.Int64
	closed                           *atomic.Bool
	maxTimeToWaitGetVM               time.Duration
	useStrict                        bool
}

// Close the scheduler
func (s *schedulerImpl) Close() error {
	s.closed.Store(true)
	close(s.vms)
	return nil
}

// NewScheduler returns a new Scheduler
func NewScheduler(opt Options) Scheduler {
	scheduler := &schedulerImpl{
		mu:                 new(sync.Mutex),
		useStrict:          opt.UseStrict,
		closed:             new(atomic.Bool),
		waiting:            new(atomic.Int64),
		unInitVMs:          new(atomic.Int64),
		maxVMs:             utils.ZeroOr(opt.MaxVMs, 1),
		initVMs:            utils.ZeroOr(opt.InitialVMs, 1),
		maxRetriesGetVM:    utils.ZeroOr(opt.MaxRetriesGetVM, DefaultMaxRetriesGetVM),
		maxTimeToWaitGetVM: utils.ZeroOr(opt.MaxTimeToWaitGetVM, DefaultMaxTimeToWaitGetVM),
	}
	scheduler.vms = make(chan VM, scheduler.maxVMs)
	for i := 0; i < scheduler.initVMs; i++ {
		scheduler.vms <- newVM(scheduler.useStrict)
	}
	scheduler.unInitVMs.Store(int64(scheduler.maxVMs - scheduler.initVMs))
	return scheduler
}

func (s *schedulerImpl) cleanIdleVMs() {
	for i := 1; i <= s.maxRetriesGetVM; i++ {
		if s.unInitVMs.Load() < int64(s.initVMs) {
			if s.waiting.Load() < int64(s.initVMs) {
				s.unInitVMs.Add(1)
				_ = <-s.vms
			}
			time.Sleep(DefaultMaxTimeToWaitGetVM)
		}
	}
}

// Get the VM
func (s *schedulerImpl) Get() (VM, error) {
	timer := time.NewTimer(s.maxTimeToWaitGetVM)
	s.waiting.Add(1)

	defer func() {
		s.waiting.Add(-1)
		go s.cleanIdleVMs()
	}()

	for i := 1; i <= s.maxRetriesGetVM; i++ {
		select {
		case vm, ok := <-s.vms:
			if !ok {
				return nil, ErrSchedulerClosed
			}
			return vm, nil
		case <-timer.C:
			if s.unInitVMs.Load() > 0 {
				s.unInitVMs.Add(-1)
				return newVM(s.useStrict), nil
			}
			logger.Warnf("could not get VM in %v", time.Duration(i)*s.maxTimeToWaitGetVM)
			timer.Reset(s.maxTimeToWaitGetVM)
		}
	}
	return nil, fmt.Errorf("could not get VM in %v",
		time.Duration(s.maxRetriesGetVM)*s.maxTimeToWaitGetVM)
}

// Release the VM
func (s *schedulerImpl) Release(vm VM) {
	if s.closed.Load() {
		return
	}

	s.vms <- vm
}
