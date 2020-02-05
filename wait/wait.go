package wait

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Option to be passed to Wait to customize the resulting instance.
type Option func(*options)

type options struct {
	clock Clock
	ctx   context.Context
}

// WithClock sets the sleeper on the options
func WithClock(clock Clock) Option {
	return func(options *options) {
		if clock != nil {
			options.clock = clock
		}
	}
}

// WithContext sets the sleeper on the options
func WithContext(ctx context.Context) Option {
	return func(options *options) {
		if ctx != nil {
			options.ctx = ctx
		}
	}
}

// Create a options instance with default values.
func newOptions() *options {
	return &options{
		clock: wallClock{},
		ctx:   context.Background(),
	}
}

// Clock represents the passage of time in a way that
// can be faked out for tests.
type Clock interface {
	// After waits for the duration to elapse and then sends the current time
	// on the returned channel.
	// It is equivalent to NewTimer(d).C.
	// The underlying Timer is not recovered by the garbage collector
	// until the timer fires. If efficiency is a concern, use NewTimer
	// instead and call Timer.Stop if the timer is no longer needed.
	After(d time.Duration) <-chan time.Time
}

// Wait represents an abstraction for waiting for a given function to be called
// and if it doesn't complete within a given time frame, an error is called and
// the context notes that it's done.
//
// This is useful when waiting upon a given thing to happen, but don't mind
// blocking for that to happen.
func Wait(fn func(context.Context), timeout time.Duration, options ...Option) error {
	opts := newOptions()
	for _, option := range options {
		option(opts)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan struct{})
	go func() {
		fn(ctx)
		ch <- struct{}{}
	}()

	select {
	case <-ch:
		return nil
	case <-opts.ctx.Done():
		return nil
	case <-opts.clock.After(timeout):
		return errors.Errorf("timed out waiting for completion")
	}
}

// wallClock implements a Clock in terms of a standard time function
type wallClock struct{}

func (wallClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
