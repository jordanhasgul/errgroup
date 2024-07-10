package errgroup_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jordanhasgul/errgroup"
	"github.com/jordanhasgul/multierr"
	"github.com/stretchr/testify/require"
)

func TestGroup_Go(t *testing.T) {
	const (
		numFuncs int = 5
		maxFuncs int = 3
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
		require.Equal(t, numFuncs, e.Len())
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
		}

		for range maxFuncs {
			barrier <- struct{}{}
		}

		err := eg.Wait()
		require.Error(t, err)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)
		require.Equal(t, maxFuncs, e.Len())
	})

	t.Run("with cancel", func(t *testing.T) {
		//t.Parallel()

		var (
			ctx, configurer = errgroup.WithCancel(context.Background())
			eg              = errgroup.New(configurer)
		)
		for i := range numFuncs {
			err := eg.Go(func() error {
				return fmt.Errorf("error %d", i)
			})
			require.NoError(t, err)
		}

		err := eg.Wait()
		require.Error(t, err)
		require.ErrorIs(t, ctx.Err(), context.Canceled)

		var e *multierr.Error
		require.ErrorAs(t, err, &e)
		require.Equal(t, 1, e.Len())
	})
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
