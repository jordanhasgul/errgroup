package errgroup_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jordanhasgul/errgroup"
	"github.com/jordanhasgul/multierr"
	"github.com/stretchr/testify/require"
)

func TestGroup_Go(t *testing.T) {
	const numGoroutines = 1 << 4

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
	require.Equal(t, numGoroutines, e.Len())
}

func TestGroup_GoWithCancel(t *testing.T) {
	const numGoroutines = 1 << 4

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

	err = eg.Go(func() error {
		return errors.New("another error")
	})
	require.Error(t, err)

	var ce *errgroup.CancelError
	require.ErrorAs(t, err, &ce)
}

type testRunner struct {
	runner errgroup.Runner
	runs   int
}

func (r *testRunner) Run(f func()) error {
	return r.runner.Run(func() {
		defer func() {
			r.runs++
		}()
		f()
	})
}

func (r *testRunner) Runs() int {
	return r.runs
}

func TestGroup_GoWithRunner(t *testing.T) {
	const numGoroutines = 1 << 4

	var (
		r = &testRunner{
			runner: &errgroup.GoRunner{},
			runs:   0,
		}
		rc = errgroup.WithRunner(r)
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
