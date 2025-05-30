package errgroup_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/jordanhasgul/errgroup"
	"github.com/jordanhasgul/multierr"
	"github.com/stretchr/testify/require"
)

var numGoroutines = 4 * runtime.NumCPU()

func TestGroup_Go(t *testing.T) {
	t.Run("all goroutines succeed so return a nil error", func(t *testing.T) {
		var eg errgroup.Group
		for range numGoroutines {
			err := eg.Go(func() error {
				return nil
			})
			require.NoError(t, err)
		}

		err := eg.Wait()
		require.NoError(t, err)
	})

	t.Run("at least 1 goroutine fails so return a non-nil error", func(t *testing.T) {
		var eg errgroup.Group
		for i := range numGoroutines {
			err := eg.Go(func() error {
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		err := eg.Wait()
		require.Error(t, err)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)
		require.Equal(t, e.Len(), numGoroutines)
	})
}

func TestGroup_GoWithCancel(t *testing.T) {
	t.Run("all goroutines succeed so return a nil error", func(t *testing.T) {
		var (
			ctx   = context.Background()
			_, cc = errgroup.WithCancel(ctx)
			eg    = errgroup.New(cc)
		)
		for range numGoroutines {
			err := eg.Go(func() error {
				return nil
			})
			require.NoError(t, err)
		}

		err := eg.Wait()
		require.NoError(t, err)
	})

	t.Run("all goroutines succeed so group is cancelled", func(t *testing.T) {
		var (
			ctx   = context.Background()
			_, cc = errgroup.WithCancel(ctx)
			eg    = errgroup.New(cc)
		)
		for range numGoroutines {
			err := eg.Go(func() error {
				return nil
			})
			require.NoError(t, err)
		}

		_ = eg.Wait()

		err := eg.Go(func() error {
			return nil
		})
		require.Error(t, err)

		var ce *errgroup.CancelError
		require.ErrorAs(t, err, &ce)
	})

	t.Run("at least 1 goroutine fails so return a non-nil error", func(t *testing.T) {
		var (
			ctx   = context.Background()
			_, cc = errgroup.WithCancel(ctx)
			eg    = errgroup.New(cc)

			barrier = make(chan struct{})
		)
		for i := range numGoroutines {
			err := eg.Go(func() error {
				barrier <- struct{}{}
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		for range numGoroutines {
			_ = <-barrier
		}

		err := eg.Wait()
		require.Error(t, err)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)
		require.Equal(t, 1, e.Len())
	})

	t.Run("at least 1 goroutine fails so group is cancelled", func(t *testing.T) {
		var (
			ctx   = context.Background()
			_, cc = errgroup.WithCancel(ctx)
			eg    = errgroup.New(cc)

			barrier = make(chan struct{})
		)
		for i := range numGoroutines {
			err := eg.Go(func() error {
				barrier <- struct{}{}
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		for range numGoroutines {
			_ = <-barrier
		}

		_ = eg.Wait()

		err := eg.Go(func() error {
			return errors.New("another error")
		})
		require.Error(t, err)

		var ce *errgroup.CancelError
		require.ErrorAs(t, err, &ce)
	})
}

type testRunner struct {
	runs int
}

func (r *testRunner) Run(f func()) error {
	defer func() {
		r.runs++
	}()
	f()
	return nil
}

func (r *testRunner) Runs() int {
	return r.runs
}

func TestGroup_GoWithRunner(t *testing.T) {
	var (
		r testRunner

		rc = errgroup.WithRunner(&r)
		eg = errgroup.New(rc)
	)
	for i := range numGoroutines {
		err := eg.Go(func() error {
			return fmt.Errorf("error %d", i)
		})
		require.NoError(t, err)
	}

	err := eg.Wait()
	require.Error(t, err)

	var e *multierr.Error
	require.ErrorAs(t, err, &e)
	require.Equal(t, numGoroutines, e.Len())
	require.Equal(t, numGoroutines, r.Runs())
}

func BenchmarkGroup_Go(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	var (
		eg errgroup.Group
		f  = func() error {
			return nil
		}
	)
	for range b.N {
		_ = eg.Go(f)
	}
	_ = eg.Wait()
}
