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
	defaultScheduler atomic.Value
	ErrVMPoolClosed  = errors.New("runtime pool is closed")
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
	mu                                          *sync.Mutex
	vms                                         chan VM
	initVMs, activeVMs, maxVMs, maxRetriesGetVM int
	maxTimeToWaitGetVM                          time.Duration
	useStrict                                   bool
}

// Close the scheduler
func (s *schedulerImpl) Close() error {
	close(s.vms)
	return nil
}

// NewScheduler returns a new Scheduler
func NewScheduler(opt Options) Scheduler {
	scheduler := &schedulerImpl{
		mu:                 new(sync.Mutex),
		vms:                make(chan VM, opt.InitialVMs),
		useStrict:          opt.UseStrict,
		initVMs:            utils.ZeroOr(opt.InitialVMs, 1),
		maxVMs:             utils.ZeroOr(opt.MaxVMs, 1),
		maxRetriesGetVM:    utils.ZeroOr(opt.MaxRetriesGetVM, DefaultMaxRetriesGetVM),
		maxTimeToWaitGetVM: utils.ZeroOr(opt.MaxTimeToWaitGetVM, DefaultMaxTimeToWaitGetVM),
	}
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	for i := 0; i < opt.InitialVMs; i++ {
		scheduler.vms <- newVM(opt.UseStrict)
		scheduler.activeVMs++
	}
	return scheduler
}

// Get the VM
func (s *schedulerImpl) Get() (VM, error) {
	timer := time.NewTimer(s.maxTimeToWaitGetVM)
	for i := 1; i <= s.maxRetriesGetVM; i++ {
		select {
		case vm := <-s.vms:
			return vm, nil
		case <-timer.C:
			s.mu.Lock()
			if s.activeVMs < s.maxVMs {
				vm := newVM(s.useStrict)
				s.activeVMs++
				s.mu.Unlock()
				return vm, nil
			} else {
				logger.Warnf("could not get VM in %v",
					time.Duration(i)*s.maxTimeToWaitGetVM)
			}
			timer.Reset(DefaultMaxTimeToWaitGetVM)
			s.mu.Unlock()
		}
	}
	return nil, fmt.Errorf("could not get VM in %v",
		time.Duration(s.maxRetriesGetVM)*s.maxTimeToWaitGetVM)
}

// Release the VM
func (s *schedulerImpl) Release(vm VM) {
	s.mu.Lock()
	s.vms <- vm
	if s.activeVMs > s.initVMs {
		s.activeVMs--
	}
	s.mu.Unlock()
}
