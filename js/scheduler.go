package js

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"log/slog"

	"github.com/shiroyk/ski"
)

const (
	// DefaultMaxTimeToWaitGetVM default retries time
	DefaultMaxTimeToWaitGetVM = 500 * time.Millisecond
	// DefaultMaxRetriesGetVM default retries times
	DefaultMaxRetriesGetVM = 3
)

var (
	_scheduler = new(atomic.Value)
	// ErrSchedulerClosed the scheduler is closed error
	ErrSchedulerClosed = errors.New("scheduler is closed")
)

func init() {
	ski.Register("js", js)
	_scheduler.Store(NewScheduler(SchedulerOptions{
		MaxVMs: uint(runtime.GOMAXPROCS(0)),
		Loader: NewModuleLoader(),
	}))
}

// SetScheduler set the default Scheduler
func SetScheduler(scheduler Scheduler) { _scheduler.Store(scheduler) }

// GetScheduler get the default Scheduler
func GetScheduler() Scheduler { return _scheduler.Load().(Scheduler) }

// Scheduler the VM scheduler
type Scheduler interface {
	// Get the VM
	Get() (VM, error)
	// Shrink the available VM
	Shrink()
	// Loader the ModuleLoader
	Loader() ModuleLoader
	// Close the scheduler
	Close() error
}

// SchedulerOptions options
type SchedulerOptions struct {
	InitialVMs         uint          `yaml:"initial-vms" json:"initialVMs"`
	MaxVMs             uint          `yaml:"max-vms" json:"maxVMs"`
	MaxRetriesGetVM    uint          `yaml:"max-retries-get-vm" json:"maxRetriesGetVM"`
	MaxTimeToWaitGetVM time.Duration `yaml:"max-time-to-wait-get-vm" json:"maxTimeToWaitGetVM"`
	Loader             ModuleLoader  `yaml:"-"` // module loader
	VMOptions          []Option      `yaml:"-"` // options for NewVM
}

// NewScheduler returns a new Scheduler
func NewScheduler(opt SchedulerOptions) Scheduler {
	s := &schedulerImpl{
		closed:             new(atomic.Bool),
		unInitVMs:          new(atomic.Int32),
		maxVMs:             opt.MaxVMs,
		maxRetriesGetVM:    opt.MaxRetriesGetVM,
		maxTimeToWaitGetVM: opt.MaxTimeToWaitGetVM,
		loader:             opt.Loader,
	}
	if s.maxVMs == 0 {
		s.maxVMs = 1
	}
	if s.maxRetriesGetVM == 0 {
		s.maxRetriesGetVM = DefaultMaxRetriesGetVM
	}
	if s.maxTimeToWaitGetVM == 0 {
		s.maxTimeToWaitGetVM = DefaultMaxTimeToWaitGetVM
	}
	if s.loader == nil {
		slog.Warn("js.ModuleLoader not provided, require and module will not working")
		s.loader = emptyLoader{}
	}
	s.maxVMs = max(s.maxVMs, opt.InitialVMs)
	s.vms = make(chan VM, s.maxVMs)
	s.vmOpt = append(opt.VMOptions,
		func(vm *vmImpl) {
			vm.release = func() { s.release(vm) }
		}, WithModuleLoader(opt.Loader))
	for i := uint(0); i < opt.InitialVMs; i++ {
		s.vms <- NewVM(s.vmOpt...)
	}
	s.unInitVMs.Store(int32(s.maxVMs - opt.InitialVMs))
	return s
}

type schedulerImpl struct {
	vms                     chan VM
	maxVMs, maxRetriesGetVM uint
	unInitVMs               *atomic.Int32
	closed                  *atomic.Bool
	maxTimeToWaitGetVM      time.Duration
	loader                  ModuleLoader
	vmOpt                   []Option
}

func (s *schedulerImpl) Loader() ModuleLoader { return s.loader }

func (s *schedulerImpl) String() string {
	text, _ := s.MarshalText()
	return string(text)
}

func (s *schedulerImpl) MarshalText() ([]byte, error) {
	return json.Marshal(map[string]any{
		"available": len(s.vms),
		"max":       int(s.maxVMs),
		"unInit":    int(s.unInitVMs.Load()),
	})
}

// Close the scheduler
func (s *schedulerImpl) Close() error {
	s.closed.Store(true)
	close(s.vms)
	return nil
}

// Get the VM
func (s *schedulerImpl) Get() (VM, error) {
	if s.unInitVMs.CompareAndSwap(int32(s.maxVMs), int32(s.maxVMs-1)) {
		return NewVM(s.vmOpt...), nil
	}

	timer := time.NewTimer(s.maxTimeToWaitGetVM)

	defer timer.Stop()

	for i := uint(1); i <= s.maxRetriesGetVM; i++ {
		select {
		case vm, ok := <-s.vms:
			if !ok {
				return nil, ErrSchedulerClosed
			}
			return vm, nil
		case <-timer.C:
			if s.unInitVMs.Add(-1) >= 0 {
				return NewVM(s.vmOpt...), nil
			}
			s.unInitVMs.Add(1)
			slog.Warn(fmt.Sprintf("could not get VM in %v", time.Duration(i)*s.maxTimeToWaitGetVM))
			timer.Reset(s.maxTimeToWaitGetVM)
		}
	}
	return nil, fmt.Errorf("could not get VM in %v",
		time.Duration(s.maxRetriesGetVM)*s.maxTimeToWaitGetVM)
}

func (s *schedulerImpl) Shrink() {
	if len(s.vms) == 0 {
		return
	}
	s.unInitVMs.Store(int32(s.maxVMs))
	for i := 0; i <= len(s.vms); i++ {
		_ = <-s.vms
	}
}

// Release the VM
func (s *schedulerImpl) release(vm VM) {
	if s.closed.Load() {
		return
	}

	s.vms <- vm
}
