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
	"golang.org/x/exp/slog"
)

const (
	MaxTimeToWaitGetVM = 500 * time.Millisecond
	MaxRetriesGetVM    = 3
)

var (
	defaultScheduler atomic.Value
	ErrVMPoolClosed  = errors.New("runtime pool is closed")
)

func init() {
	defaultScheduler.Store(NewScheduler(Options{InitialCap: 2, MaxCap: runtime.NumCPU()}))
}

// SetScheduler makes s the default Scheduler.
func SetScheduler(s Scheduler) {
	defaultScheduler.Store(s)
}

// GetScheduler returns the default Scheduler.
func GetScheduler() Scheduler {
	return defaultScheduler.Load().(Scheduler)
}

func RunString(ctx context.Context, script string) (goja.Value, error) {
	tr, err := GetScheduler().Get()
	if err != nil {
		return nil, err
	}
	return tr.RunString(ctx, script)
}

func Run(ctx context.Context, p Program) (goja.Value, error) {
	tr, err := GetScheduler().Get()
	if err != nil {
		return nil, err
	}
	return tr.Run(ctx, p)
}

type Scheduler interface {
	Get() (VM, error)
	Release(VM)
	Close() error
}

type Options struct {
	InitialCap, MaxCap, MaxRetriesGetVM int
	UseStrict                           bool
}

type schedulerImpl struct {
	mu        *sync.Mutex
	vms       chan VM
	activeVMs *atomic.Uint64
	opt       Options
}

func (c *schedulerImpl) Close() error {
	close(c.vms)
	return nil
}

func NewScheduler(opt Options) Scheduler {
	return &schedulerImpl{
		mu:        new(sync.Mutex),
		vms:       make(chan VM, opt.InitialCap),
		activeVMs: new(atomic.Uint64),
		opt:       opt,
	}
}

func (c *schedulerImpl) Get() (VM, error) {
	if c.activeVMs.Load() == 0 {
		return newVM(c.opt.UseStrict), nil
	}
	for i := 1; i <= MaxRetriesGetVM; i++ {
		select {
		case rt := <-c.vms:
			return rt, nil
		case <-time.After(MaxTimeToWaitGetVM):
			slog.Warn("Could not get a VM from the buffer for %s", time.Duration(i)*MaxTimeToWaitGetVM)
		}
	}
	return nil, fmt.Errorf(
		"could not get a VM from the buffer in %s",
		MaxRetriesGetVM*MaxTimeToWaitGetVM,
	)
}

func (c *schedulerImpl) Release(vm VM) {
	c.vms <- vm
	c.activeVMs.Add(+1)
}
