package js

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventLoop(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		loop := NewEventLoop()
		var i int
		f := func() error { i++; return nil }
		require.NoError(t, loop.Start(f))
		assert.Equal(t, 1, i)
		require.NoError(t, loop.Start(f))
		assert.Equal(t, 2, i)
	})

	t.Run("basic execution", func(t *testing.T) {
		loop := NewEventLoop()
		var result int

		err := loop.Start(func() error {
			enqueue := loop.EnqueueJob()
			enqueue(func() error { result = 42; return nil })
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("sequential tasks", func(t *testing.T) {
		loop := NewEventLoop()
		var sequence []int

		err := loop.Start(func() error {
			loop.EnqueueJob()(func() error {
				sequence = append(sequence, 1)
				return nil
			})
			loop.EnqueueJob()(func() error {
				sequence = append(sequence, 2)
				return nil
			})
			loop.EnqueueJob()(func() error {
				sequence = append(sequence, 3)
				return nil
			})
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, sequence)
	})

	t.Run("nested tasks", func(t *testing.T) {
		loop := NewEventLoop()
		var sequence []int

		err := loop.Start(func() error {
			loop.EnqueueJob()(func() error {
				sequence = append(sequence, 1)
				loop.EnqueueJob()(func() error {
					sequence = append(sequence, 2)
					loop.EnqueueJob()(func() error {
						sequence = append(sequence, 3)
						return nil
					})
					return nil
				})
				return nil
			})
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, sequence)
	})

	t.Run("cleanup order", func(t *testing.T) {
		loop := NewEventLoop()
		var sequence []int

		err := loop.Start(func() error {
			loop.Cleanup(func() { sequence = append(sequence, 3) })
			loop.Cleanup(func() { sequence = append(sequence, 2) })
			loop.Cleanup(func() { sequence = append(sequence, 1) })
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, []int{3, 2, 1}, sequence)
	})

	t.Run("concurrent tasks", func(t *testing.T) {
		loop := NewEventLoop()
		results := make(map[int]bool)
		const goroutines = 100

		err := loop.Start(func() error {
			for i := range goroutines {
				enqueue := loop.EnqueueJob()
				go func(i int) {
					time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)))
					enqueue(func() error { results[i] = true; return nil })
				}(i)
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, goroutines, len(results))
		for i := range goroutines {
			assert.True(t, results[i])
		}
	})

	t.Run("reuse enqueue", func(t *testing.T) {
		loop := NewEventLoop()

		err := loop.Start(func() error {
			enqueue := loop.EnqueueJob()
			enqueue(func() error { return nil })
			assert.PanicsWithValue(t, "Enqueue already called", func() {
				enqueue(func() error { return nil })
			})
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("stop during execution", func(t *testing.T) {
		loop := NewEventLoop()
		var cleanupCalled bool

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		go func() {
			_ = loop.Start(func() error {
				loop.EnqueueJob()
				loop.Cleanup(func() { cleanupCalled = true })
				return nil
			})
		}()

		<-ctx.Done()
		loop.Stop()
		assert.True(t, cleanupCalled)
	})

	t.Run("multiple stops", func(t *testing.T) {
		loop := NewEventLoop()
		var cleanupCount int

		_ = loop.Start(func() error {
			loop.Cleanup(func() { cleanupCount++ })
			return nil
		})

		loop.Stop()
		loop.Stop()
		loop.Stop()

		assert.Equal(t, 1, cleanupCount)
	})

	t.Run("enqueue after stop", func(t *testing.T) {
		loop := NewEventLoop()
		executed := false

		_ = loop.Start(func() error {
			enqueue := loop.EnqueueJob()
			loop.Stop()
			enqueue(func() error { executed = true; return nil })
			return nil
		})

		assert.False(t, executed)
	})

	t.Run("error in main task", func(t *testing.T) {
		loop := NewEventLoop()
		expectedErr := errors.New("main task error")

		err := loop.Start(func() error { return expectedErr })

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("error in enqueued task", func(t *testing.T) {
		loop := NewEventLoop()
		expectedErr := errors.New("enqueued task error")

		err := loop.Start(func() error {
			enqueue := loop.EnqueueJob()
			enqueue(func() error { return expectedErr })
			return nil
		})

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("error in nested enqueued tasks", func(t *testing.T) {
		loop := NewEventLoop()
		expectedErr := errors.New("nested error")

		err := loop.Start(func() error {
			loop.EnqueueJob()(func() error {
				loop.EnqueueJob()(func() error { return expectedErr })
				return nil
			})
			return nil
		})

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("cleanup tasks run after error", func(t *testing.T) {
		loop := NewEventLoop()
		expectedErr := errors.New("task error")
		cleanupCalled := false

		err := loop.Start(func() error {
			loop.Cleanup(func() { cleanupCalled = true })
			return expectedErr
		})

		assert.ErrorIs(t, err, expectedErr)
		assert.True(t, cleanupCalled, "cleanup should be called even after error")
	})

	t.Run("panic in enqueued task", func(t *testing.T) {
		loop := NewEventLoop()
		panicMsg := "panic in task"

		assert.PanicsWithValue(t, panicMsg, func() {
			_ = loop.Start(func() error {
				enqueue := loop.EnqueueJob()
				enqueue(func() error { panic(panicMsg) })
				return nil
			})
		})
	})

	t.Run("error after stop", func(t *testing.T) {
		loop := NewEventLoop()
		errChan := make(chan error, 1)

		go func() {
			errChan <- loop.Start(func() error {
				enqueue := loop.EnqueueJob()
				go func() {
					time.Sleep(time.Millisecond * 100)
					enqueue(func() error { return errors.New("should not execute") })
				}()
				return nil
			})
		}()

		time.Sleep(time.Millisecond * 50)
		loop.Stop()

		select {
		case err := <-errChan:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for loop to stop")
		}
	})

	t.Run("concurrent errors", func(t *testing.T) {
		loop := NewEventLoop()
		expectedErr := errors.New("concurrent error")
		const goroutines = 10

		err := loop.Start(func() error {
			for range goroutines {
				enqueue := loop.EnqueueJob()
				go func() {
					time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
					enqueue(func() error { return expectedErr })
				}()
			}
			return nil
		})

		assert.ErrorIs(t, err, expectedErr)
		unwrap, ok := err.(interface{ Unwrap() []error })
		assert.True(t, ok)
		assert.Equal(t, goroutines, len(unwrap.Unwrap()))
	})
}
