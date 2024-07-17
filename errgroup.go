// Package errgroup is a Go package that provides synchronisation and error
// propagation for goroutines running fallible functions.
package errgroup

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/jordanhasgul/multierr"
)

// Group manages the execution of goroutines that run fallible functions.
type Group struct {
	wg        sync.WaitGroup
	semaphore chan struct{}
	cancel    context.CancelFunc
	cancelled atomic.Bool

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

// Go tries to launch the fallible function f in another goroutine. If it
// could not, Go returns an error explaining why:
//
//   - A CancelError if the Group has been cancelled.
//   - A LimitError if launching f in another goroutine would cause the
//     number of goroutines managed by the Group to exceed its limit.
func (g *Group) Go(f func() error) error {
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
			g.handleError(err)
		}
	}()

	return nil
}

func (g *Group) handleError(err error) {
	g.errLock.Lock()
	defer g.errLock.Unlock()

	if !g.cancelled.Load() {
		if g.cancel != nil {
			g.cancelled.Store(true)
			g.cancel()
		}

		g.err = multierr.Append(g.err, err)
	}
}

// Wait blocks until all goroutines managed by the Group have finished
// executing and returns an error that aggregates any errors that occurred
// within each goroutine.
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

// WithCancel returns context.Context derived from ctx and a Configurer. The
// returned Configurer configures a Group to cancel the derived
// context.Context the first time a fallible function passed to Group.Go
// returns a non-nil error.
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
