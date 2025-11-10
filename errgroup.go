// Package errgroup is a Go package that provides Context cancellation, error
// propagation and synchronisation for goroutines running fallible functions.
package errgroup

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/jordanhasgul/multierr"
)

// Group manages the execution of fallible functions i.e. functions of type
// func() error.
type Group struct {
	wg     sync.WaitGroup
	runner Runner

	cancel    context.CancelFunc
	cancelled atomic.Bool

	errLock sync.Mutex
	err     error
}

// Configurer configures the behaviour of a Group.
type Configurer interface {
	configure(g *Group)
}

// New returns a new Group that has been configured by applying the supplied
// Configurer's.
func New(configurers ...Configurer) *Group {
	group := &Group{}
	for _, configurer := range configurers {
		configurer.configure(group)
	}

	return group
}

// CancelError indicates that the Group has been cancelled.
type CancelError struct{}

func (c CancelError) Error() string {
	return "group has been cancelled"
}

// Go runs f according to the semantics of the Group's Runner, or it returns
// a CancelError if the Group has been cancelled.
func (g *Group) Go(f func() error) error {
	if g.runner == nil {
		g.runner = &GoRunner{}
	}

	if g.cancelled.Load() {
		return &CancelError{}
	}

	g.wg.Add(1)
	return g.runner.Run(func() {
		defer g.wg.Done()

		err := f()
		if err != nil {
			if !g.cancelled.Load() {
				g.errLock.Lock()
				defer g.errLock.Unlock()

				g.err = multierr.Append(g.err, err)
				if g.cancel != nil {
					g.cancel()
				}
			}
		}
	})
}

// Wait blocks until the Group has run every f supplied to Group.Go and
// returns an error that aggregates any errors that occurred.
func (g *Group) Wait() error {
	g.wg.Wait()

	if g.cancel != nil {
		g.cancel()
	}

	g.errLock.Lock()
	defer g.errLock.Unlock()
	return g.err
}

// Runner runs f in a goroutine.
type Runner interface {
	Run(f func()) error
}

// GoRunner is a Runner that runs f in another goroutine that was spawned
// using the `go` keyword.
type GoRunner struct{}

func (r *GoRunner) Run(f func()) error {
	go f()
	return nil
}

type runnerConfigurer struct {
	runner Runner
}

func (c runnerConfigurer) configure(g *Group) {
	g.runner = c.runner
}

// WithRunner returns a Configurer that configures a Group to run every f
// supplied to Group.Go using the supplied Runner.
func WithRunner(runner Runner) Configurer {
	return &runnerConfigurer{runner: runner}
}

type cancelConfigurer struct {
	cancel context.CancelFunc
}

func (c cancelConfigurer) configure(group *Group) {
	group.cancel = func() {
		group.cancelled.Store(true)
		c.cancel()
	}
}

// WithCancel returns a context.Context (derived from ctx) and a Configurer. The
// returned Configurer configures a Group to cancel the derived context.Context when:
//
//   - The first time a function passed to Group.Go returns a non-nil error.
//   - The first time a call to Group.Wait returns.
func WithCancel(ctx context.Context) (context.Context, Configurer) {
	ctx, cancel := context.WithCancel(ctx)
	return ctx, &cancelConfigurer{cancel}
}
