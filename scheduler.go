package ski

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/shiroyk/ski/js"
)

const (
	// DefaultTimeoutGetVM default get js.VM timeout
	DefaultTimeoutGetVM = 500 * time.Millisecond
	// DefaultMaxRetriesGetVM default retries times
	DefaultMaxRetriesGetVM = 3
)

var (
	scheduler = new(atomic.Value)
	// ErrSchedulerClosed the scheduler was closed error
	ErrSchedulerClosed = errors.New("scheduler was closed")
)

// Metrics contains Scheduler metrics
type Metrics struct {
	Max       int `json:"max"`       // max vm size
	Idle      int `json:"idle"`      // idle vm size
	Remaining int `json:"remaining"` // remaining creatable vm size
}

// SchedulerOptions options
type SchedulerOptions struct {
	InitialVMs    uint          `yaml:"initial-vms" json:"initialVMs"`
	MaxVMs        uint          `yaml:"max-vms" json:"maxVMs"`
	GetMaxRetries uint          `yaml:"get-max-retries" json:"maxRetries"`
	GetTimeout    time.Duration `yaml:"get-timeout" json:"timeout"`
	VMOptions     []js.Option   `yaml:"-"` // options for NewVM
}

func init() {
	SetScheduler(NewScheduler(SchedulerOptions{}))
}

// SetScheduler set the default Scheduler
func SetScheduler(s Scheduler) { scheduler.Store(s) }

// GetScheduler get the default Scheduler
func GetScheduler() Scheduler { return scheduler.Load().(Scheduler) }

// Scheduler the js.VM scheduler
type Scheduler interface {
	// Get the VM1
	Get() (js.VM, error)
	// Shrink the idle VM to initial VM size
	Shrink()
	// Metrics Scheduler metrics
	Metrics() Metrics
	// Close the scheduler
	Close() error
}

// NewScheduler create a new Scheduler
func NewScheduler(opt SchedulerOptions) Scheduler {
	if opt.MaxVMs == 0 {
		opt.MaxVMs = uint(runtime.GOMAXPROCS(0))
	}
	if opt.GetMaxRetries == 0 {
		opt.GetMaxRetries = DefaultMaxRetriesGetVM
	}
	if opt.GetTimeout == 0 {
		opt.GetTimeout = DefaultTimeoutGetVM
	}

	s := &schedulerImpl{
		vms:        make(chan js.VM, opt.MaxVMs),
		maxVMs:     int32(opt.MaxVMs),
		maxRetries: opt.GetMaxRetries,
		timeout:    opt.GetTimeout,
	}
	s.opt = append(opt.VMOptions, js.WithRelease(s.release))

	s.initial = opt.InitialVMs
	if s.initial > opt.MaxVMs {
		s.initial = opt.MaxVMs
	}

	s.active.Store(int32(s.initial))
	for range s.initial {
		s.vms <- js.NewVM(s.opt...)
	}

	return s
}

type schedulerImpl struct {
	vms        chan js.VM
	active     atomic.Int32
	maxVMs     int32
	maxRetries uint
	initial    uint
	timeout    time.Duration
	closed     atomic.Bool
	opt        []js.Option
}

func (s *schedulerImpl) Get() (js.VM, error) {
	if s.closed.Load() {
		return nil, ErrSchedulerClosed
	}

	for range s.maxRetries {
		select {
		case vm := <-s.vms:
			return vm, nil
		default:
			if s.active.Add(1) > s.maxVMs {
				s.active.Add(-1) // rollback count
				select {
				case vm := <-s.vms:
					return vm, nil
				case <-time.After(s.timeout):
					continue
				}
			}
			return js.NewVM(s.opt...), nil // create new
		}
	}

	return nil, fmt.Errorf("could not get VM in %v", time.Duration(s.maxRetries)*s.timeout)
}

func (s *schedulerImpl) release(vm js.VM) {
	if s.closed.Load() {
		return
	}
	select {
	case s.vms <- vm:
	default:
	}
}

func (s *schedulerImpl) Metrics() Metrics {
	m := Metrics{
		Max:  int(s.maxVMs),
		Idle: len(s.vms),
	}
	m.Remaining = m.Max - int(s.active.Load())
	return m
}

func (s *schedulerImpl) Shrink() {
	if len(s.vms) == 0 {
		return
	}

	for range max(len(s.vms)-int(s.initial), 0) {
		select {
		case _ = <-s.vms:
			s.active.Add(-1)
		default:
		}
	}
}

func (s *schedulerImpl) Close() error {
	if s.closed.Swap(true) {
		return ErrSchedulerClosed
	}
	s.Shrink()
	close(s.vms)
	return nil
}

func (s *schedulerImpl) String() string {
	text, _ := s.MarshalText()
	return string(text)
}

func (s *schedulerImpl) MarshalText() ([]byte, error) {
	return json.Marshal(s.Metrics())
}
