package ski

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shiroyk/ski/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		s := NewScheduler(SchedulerOptions{
			InitialVMs: 2,
			MaxVMs:     5,
		})
		defer s.Close()

		metrics := s.Metrics()
		assert.Equal(t, 5, metrics.Max)
		assert.Equal(t, 2, metrics.Idle)
		assert.Equal(t, 3, metrics.Remaining)

		vm1, err := s.get()
		require.NoError(t, err)
		require.NotNil(t, vm1)

		metrics = s.Metrics()
		assert.Equal(t, 5, metrics.Max)
		assert.Equal(t, 1, metrics.Idle)
		assert.Equal(t, 3, metrics.Remaining)

		vm1.Run(context.Background(), func() error { return nil })

		metrics = s.Metrics()
		assert.Equal(t, 5, metrics.Max)
		assert.Equal(t, 2, metrics.Idle)
		assert.Equal(t, 3, metrics.Remaining)
	})

	t.Run("max VMs limit", func(t *testing.T) {
		s := NewScheduler(SchedulerOptions{
			MaxVMs:        2,
			GetMaxRetries: 2,
			GetTimeout:    100 * time.Millisecond,
		})
		defer s.Close()

		vm1, err := s.get()
		require.NoError(t, err)
		vm2, err := s.get()
		require.NoError(t, err)

		_, err = s.get()
		assert.Error(t, err)

		vm1.Run(context.Background(), func() error { return nil })

		vm3, err := s.get()
		assert.NoError(t, err)
		assert.NotNil(t, vm3)

		vm2.Run(context.Background(), func() error { return nil })
		vm3.Run(context.Background(), func() error { return nil })

		metrics := s.Metrics()
		assert.Equal(t, 2, metrics.Max)
		assert.Equal(t, 2, metrics.Idle)
		assert.Equal(t, 0, metrics.Remaining)
	})

	t.Run("concurrent operations", func(t *testing.T) {
		s := NewScheduler(SchedulerOptions{
			InitialVMs:    5,
			MaxVMs:        10,
			GetMaxRetries: 3,
			GetTimeout:    100 * time.Millisecond,
		})
		defer s.Close()

		var wg1, wg2 sync.WaitGroup
		vms := make(chan js.VM, 20)
		wg2.Add(1)

		go func() {
			defer wg2.Done()
			for vm := range vms {
				vm.Run(context.Background(), func() error { return nil })
			}
		}()

		for range 20 {
			wg1.Add(1)
			go func() {
				defer wg1.Done()
				vm, err := s.get()
				if err == nil {
					vms <- vm
				}
			}()
		}

		wg1.Wait()
		close(vms)
		wg2.Wait()

		metrics := s.Metrics()
		assert.Equal(t, 10, metrics.Remaining+metrics.Idle)
	})

	t.Run("shrink operation", func(t *testing.T) {
		s := NewScheduler(SchedulerOptions{
			InitialVMs: 2,
			MaxVMs:     4,
		})
		defer s.Close()

		vm1, _ := s.get()
		vm2, _ := s.get()
		vm3, _ := s.get()

		vm1.Run(context.Background(), func() error { return nil })
		vm2.Run(context.Background(), func() error { return nil })
		vm3.Run(context.Background(), func() error { return nil })

		metrics := s.Metrics()
		assert.Equal(t, 3, metrics.Idle)
		assert.Equal(t, 1, metrics.Remaining)

		s.Shrink()

		metrics = s.Metrics()
		assert.Equal(t, 2, metrics.Idle)
		assert.Equal(t, 2, metrics.Remaining)

		vm, err := s.get()
		assert.NoError(t, err)
		assert.NotNil(t, vm)
		vm.Run(context.Background(), func() error { return nil })
	})

	t.Run("close operation", func(t *testing.T) {
		s := NewScheduler(SchedulerOptions{
			InitialVMs: 2,
			MaxVMs:     5,
		})

		vm, err := s.get()
		require.NoError(t, err)

		err = s.Close()
		assert.NoError(t, err)

		_, err = s.get()
		assert.ErrorIs(t, err, ErrSchedulerClosed)

		assert.NotPanics(t, func() {
			vm.Run(context.Background(), func() error { return nil })
		})

		err = s.Close()
		assert.ErrorIs(t, err, ErrSchedulerClosed)
	})

	t.Run("metrics accuracy", func(t *testing.T) {
		s := NewScheduler(SchedulerOptions{
			InitialVMs: 3,
			MaxVMs:     5,
		})
		defer s.Close()

		metrics := s.Metrics()
		assert.Equal(t, 5, metrics.Max)
		assert.Equal(t, 3, metrics.Idle)
		assert.Equal(t, 2, metrics.Remaining)

		vms := make([]js.VM, 3)
		for i := range 3 {
			vm, err := s.get()
			require.NoError(t, err)
			vms[i] = vm
		}

		metrics = s.Metrics()
		assert.Equal(t, 5, metrics.Max)
		assert.Equal(t, 0, metrics.Idle)
		assert.Equal(t, 2, metrics.Remaining)

		for _, vm := range vms {
			vm.Run(context.Background(), func() error { return nil })
		}

		metrics = s.Metrics()
		assert.Equal(t, 5, metrics.Max)
		assert.Equal(t, 3, metrics.Idle)
		assert.Equal(t, 2, metrics.Remaining)
	})
}
