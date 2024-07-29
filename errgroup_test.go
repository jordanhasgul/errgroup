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
	const (
		numFuncs uint = 5
		maxFuncs uint = 3
	)

	t.Run("default", func(t *testing.T) {
		t.Parallel()

		var eg errgroup.Group
		for i := range numFuncs {
			err := eg.Go(func() error {
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		err := eg.Wait()
		require.Error(t, err)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)

		numErrors := uint(e.Len())
		require.Equal(t, numFuncs, numErrors)
	})

	t.Run("with limit", func(t *testing.T) {
		t.Parallel()

		var (
			eg = errgroup.New(
				errgroup.WithLimit(maxFuncs),
			)
			barrier = make(chan struct{}, maxFuncs)
		)
		for i := range maxFuncs {
			err := eg.Go(func() error {
				_ = <-barrier
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		for i := range numFuncs - maxFuncs {
			err := eg.Go(func() error {
				return fmt.Errorf("error %d", i)
			})
			require.Error(t, err)

			var le *errgroup.LimitError
			require.ErrorAs(t, err, &le)
		}

		for range maxFuncs {
			barrier <- struct{}{}
		}

		err := eg.Wait()
		require.Error(t, err)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)

		numErrors := uint(e.Len())
		require.Equal(t, maxFuncs, numErrors)
	})

	t.Run("with cancel", func(t *testing.T) {
		t.Parallel()

		var (
			_, configurer = errgroup.WithCancel(context.Background())
			eg            = errgroup.New(configurer)
		)
		for i := range numFuncs {
			err := eg.Go(func() error {
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		err := eg.Wait()
		require.Error(t, err)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)
		require.Equal(t, 1, e.Len())

		err = eg.Go(func() error {
			return fmt.Errorf("error %d", numFuncs)
		})
		require.Error(t, err)

		var ce *errgroup.CancelError
		require.ErrorAs(t, err, &ce)
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
