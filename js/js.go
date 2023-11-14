package js

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat"
)

const (
	// DefaultMaxTimeToWaitGetVM default retries time
	DefaultMaxTimeToWaitGetVM = 500 * time.Millisecond
	// DefaultMaxRetriesGetVM default retries times
	DefaultMaxRetriesGetVM = 3
)

var (
	defaultScheduler Scheduler
	defaultOptions   = Options{InitialVMs: 2, MaxVMs: runtime.GOMAXPROCS(0)}
	schedulerOnce    sync.Once
	// ErrSchedulerClosed the scheduler is closed error
	ErrSchedulerClosed = errors.New("scheduler is closed")
)

// SetOptions set the default js Options.
func SetOptions(opt Options) {
	defaultOptions = opt
}

// GetScheduler returns the default Scheduler.
func GetScheduler() Scheduler {
	schedulerOnce.Do(func() {
		defaultScheduler = NewScheduler(defaultOptions)
	})
	return defaultScheduler
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
func Run(ctx context.Context, p Program) (goja.Value, error) {
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
	InitialVMs         int           `yaml:"initial-vms"`
	MaxVMs             int           `yaml:"max-vms"`
	MaxRetriesGetVM    int           `yaml:"max-retries-get-vm"`
	MaxTimeToWaitGetVM time.Duration `yaml:"max-time-to-wait-get-vm"`
	ModulePath         []string      `yaml:"module-path"`
}

type schedulerImpl struct {
	mu                               *sync.Mutex
	vms                              chan VM
	initVMs, maxVMs, maxRetriesGetVM int
	unInitVMs                        *atomic.Int64
	closed                           *atomic.Bool
	maxTimeToWaitGetVM               time.Duration
	modulePath                       []string
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
		closed:             new(atomic.Bool),
		unInitVMs:          new(atomic.Int64),
		maxVMs:             cloudcat.ZeroOr(opt.MaxVMs, 1),
		initVMs:            cloudcat.ZeroOr(opt.InitialVMs, 1),
		maxRetriesGetVM:    cloudcat.ZeroOr(opt.MaxRetriesGetVM, DefaultMaxRetriesGetVM),
		maxTimeToWaitGetVM: cloudcat.ZeroOr(opt.MaxTimeToWaitGetVM, DefaultMaxTimeToWaitGetVM),
		modulePath:         opt.ModulePath,
	}
	scheduler.vms = make(chan VM, scheduler.maxVMs)
	for i := 0; i < scheduler.initVMs; i++ {
		scheduler.vms <- NewVM(scheduler.modulePath...)
	}
	scheduler.unInitVMs.Store(int64(scheduler.maxVMs - scheduler.initVMs))
	return scheduler
}

// Get the VM
func (s *schedulerImpl) Get() (VM, error) {
	timer := time.NewTimer(s.maxTimeToWaitGetVM)

	defer func() {
		timer.Stop()
	}()

	for i := 1; i <= s.maxRetriesGetVM; i++ {
		select {
		case vm, ok := <-s.vms:
			if !ok {
				return nil, ErrSchedulerClosed
			}
			return vm, nil
		case <-timer.C:
			if s.unInitVMs.Add(-1) >= 0 {
				return NewVM(s.modulePath...), nil
			}
			s.unInitVMs.Add(1)
			slog.Warn(fmt.Sprintf("could not get VM in %v", time.Duration(i)*s.maxTimeToWaitGetVM))
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
