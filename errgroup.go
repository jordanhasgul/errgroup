// Package errgroup is a Go package that provides Context cancellation, error propagation and 
// synchronisation for goroutines running fallible functions.
package errgroup

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/jordanhasgul/multierr"
)

// Group manages the execution of goroutines that run functions of type
// func() error.
type Group struct {
	semaphore chan struct{}
	wg        sync.WaitGroup
	cancelled atomic.Bool
	cancel    context.CancelFunc

	errLock sync.Mutex
	err     error
}

// Configurer is implemented by any type that has a configure method. The
// configure method is used to configure the behaviour of a Group.
type Configurer interface {
	configure(*Group)
}

// New returns a new Group that has been configured by applying any supplied
// configurers.
func New(configurers ...Configurer) *Group {
	group := &Group{}
	for _, configurer := range configurers {
		configurer.configure(group)
	}

	return group
}

// LimitError indicates that a Group has reached its limit.
type LimitError struct {
	limit int
}

var _ error = (*LimitError)(nil)

func (e LimitError) Error() string {
	errorString := "group has reached the limit of %d goroutines"
	return fmt.Sprintf(errorString, e.limit)
}

// CancelError indicates that a Group has been cancelled.
type CancelError struct{}

var _ error = (*CancelError)(nil)

func (c CancelError) Error() string {
	return "group has been cancelled"
}

// Go launch f in another goroutine. It blocks until the new goroutine can
// be added without causing number of goroutines managed by the Group to
// exceed its limit. If the Group has been cancelled, a CancelError is
// returned.
func (g *Group) Go(f func() error) error {
	if g.cancelled.Load() {
		return &CancelError{}
	}

	if g.semaphore != nil {
		g.semaphore <- struct{}{}
	}

	g.doGo(f)
	return nil
}

// TryGo tries to launch f in another goroutine. If it could not, TryGo
// returns an error explaining why:
//
//   - A CancelError if the Group has been cancelled.
//   - A LimitError if launching f in another goroutine would cause the
//     number of goroutines managed by the Group to exceed its limit.
func (g *Group) TryGo(f func() error) error {
	if g.cancelled.Load() {
		return &CancelError{}
	}

	if g.semaphore != nil {
		select {
		case g.semaphore <- struct{}{}:
		default:
			return &LimitError{
				limit: cap(g.semaphore),
			}
		}
	}

	g.doGo(f)
	return nil
}

func (g *Group) doGo(f func() error) {
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
			if !g.cancelled.Load() {
				if g.cancel != nil {
					g.cancel()
				}

				g.errLock.Lock()
				defer g.errLock.Unlock()
				g.err = multierr.Append(g.err, err)
			}
		}
	}()
}

// Wait blocks until all goroutines managed by the Group have finished
// executing and returns an error that aggregates any errors that occurred
// within each goroutine.
func (g *Group) Wait() error {
	g.wg.Wait()

	if g.cancel != nil {
		g.cancel()
	}

	g.errLock.Lock()
	defer g.errLock.Unlock()
	return g.err
}

type cancelConfigurer struct {
	cancel context.CancelFunc
}

var _ Configurer = (*cancelConfigurer)(nil)

func (c cancelConfigurer) configure(group *Group) {
	group.cancel = func() {
		group.cancelled.Store(true)
		c.cancel()
	}
}

// WithCancel returns context.Context derived from ctx and a Configurer. The
// returned Configurer configures a Group to cancel the derived
// context.Context when:
//
//   - The first time a function passed to Group.Go returns a non-nil error.
//   - The first time a call to Group.Wait returns.
func WithCancel(ctx context.Context) (context.Context, Configurer) {
	ctx, cancel := context.WithCancel(ctx)
	return ctx, &cancelConfigurer{cancel}
}

type limitConfigurer struct {
	limit uint
}

var _ Configurer = (*limitConfigurer)(nil)

func (c limitConfigurer) configure(group *Group) {
	group.semaphore = make(chan struct{}, c.limit)
}

// WithLimit returns a Configurer that configures a Group to keep the number
// of goroutines managed by the Group at or below the limit.
func WithLimit(limit uint) Configurer {
	return &limitConfigurer{limit: limit}
}
