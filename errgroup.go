package errgroup

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/jordanhasgul/multierr"
)

type Group struct {
	semaphore chan struct{}
	wg        sync.WaitGroup

	cancelled atomic.Bool
	cancel    context.CancelFunc

	errLock sync.Mutex
	err     error
}

type Configurer interface {
	configure(*Group)
}

func New(configurers ...Configurer) *Group {
	group := &Group{}
	for _, configurer := range configurers {
		configurer.configure(group)
	}

	return group
}

type LimitError struct {
	limit int
}

var _ error = (*LimitError)(nil)

func (e LimitError) Error() string {
	errorString := "errgroup has reached the limit of %d goroutines"
	return fmt.Sprintf(errorString, e.limit)
}

func (g *Group) Go(f func() error) error {
	if g.semaphore != nil {
		select {
		case g.semaphore <- struct{}{}:
		default:
			return &LimitError{
				limit: cap(g.semaphore),
			}
		}
	}

	g.wg.Add(1)
	go func() {
		defer func() {
			g.wg.Done()

			if g.semaphore != nil {
				_ = <-g.semaphore
			}
		}()

		err := f()
		if err != nil {
			// https://en.wikipedia.org/wiki/Double-checked_locking
			cancelled := g.cancelled.Load()
			if !cancelled {
				g.errLock.Lock()
				defer g.errLock.Unlock()

				cancelled = g.cancelled.Load()
				if !cancelled {
					if g.cancel != nil {
						g.cancel()
						g.cancelled.Store(true)
					}

					g.err = multierr.Append(g.err, err)
				}
			}
		}
	}()

	return nil
}

func (g *Group) Wait() error {
	g.wg.Wait()

	g.errLock.Lock()
	defer g.errLock.Unlock()

	return g.err
}

type cancelConfigurer struct {
	cancel context.CancelFunc
}

var _ Configurer = (*cancelConfigurer)(nil)

func (c cancelConfigurer) configure(group *Group) {
	group.cancel = c.cancel
}

func WithCancel(ctx context.Context) (context.Context, Configurer) {
	ctx, cancel := context.WithCancel(ctx)
	return ctx, &cancelConfigurer{cancel}
}

type limitConfigurer struct {
	limit int
}

var _ Configurer = (*limitConfigurer)(nil)

func (c limitConfigurer) configure(group *Group) {
	if c.limit < 0 {
		group.semaphore = nil
		return
	}

	group.semaphore = make(chan struct{}, c.limit)
}

func WithLimit(limit int) Configurer {
	return &limitConfigurer{limit: limit}
}
