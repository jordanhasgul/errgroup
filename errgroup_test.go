package errgroup_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jordanhasgul/errgroup"
	"github.com/jordanhasgul/multierr"
	"github.com/stretchr/testify/require"
)

func TestGroup_Go(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("with cancel", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("with limit", func(t *testing.T) {
		t.Parallel()

		const (
			maxGoroutines = 1 << 4
			numGoroutines = 1 << 8
		)

		var (
			eg = errgroup.New(
				errgroup.WithLimit(maxGoroutines),
			)
			active atomic.Int32
		)
		for range numGoroutines {
			err := eg.Go(func() error {
				n := active.Add(1)
				defer active.Add(-1)
				if n > maxGoroutines {
					return fmt.Errorf("too many goroutines - got: %d, want: %d", n, maxGoroutines)
				}

				return nil
			})
			require.NoError(t, err)
		}

		err := eg.Wait()
		require.NoError(t, err)
	})
}

func TestGroup_TryGo(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Parallel()

		const numGoroutines = 1 << 4

		var eg errgroup.Group
		for i := range numGoroutines {
			err := eg.TryGo(func() error {
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		err := eg.Wait()
		require.Error(t, err)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)
		require.Equal(t, numGoroutines, e.Len())
	})

	t.Run("with cancel", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("with limit", func(t *testing.T) {
		t.Parallel()

		const (
			maxGoroutines = 1 << 4
			numGoroutines = 1 << 8
		)

		var (
			eg = errgroup.New(
				errgroup.WithLimit(maxGoroutines),
			)
			barrier = make(chan struct{}, maxGoroutines)
		)
		for i := range maxGoroutines {
			err := eg.TryGo(func() error {
				_ = <-barrier
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		for i := range numGoroutines - maxGoroutines {
			err := eg.TryGo(func() error {
				return fmt.Errorf("error %d", i)
			})
			require.Error(t, err)

			var le *errgroup.LimitError
			require.ErrorAs(t, err, &le)
		}

		for range maxGoroutines {
			barrier <- struct{}{}
		}

		err := eg.Wait()
		require.Error(t, err)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)
		require.Equal(t, maxGoroutines, e.Len())
	})
}

func BenchmarkGroup_Go(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	var (
		eg errgroup.Group

		err = errors.New("some error")
		f   = func() error { return err }
	)
	for range b.N {
		_ = eg.Go(f)
	}
	_ = eg.Wait()
}

func BenchmarkGroup_TryGo(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	var (
		eg errgroup.Group

		err = errors.New("some error")
		f   = func() error { return err }
	)
	for range b.N {
		_ = eg.TryGo(f)
	}
	_ = eg.Wait()
}
