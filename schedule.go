package task

import (
	"time"

	"github.com/pkg/errors"
)

// Schedule captures the signature of a schedule function.
//
// It should return the amount of time to wait before triggering the next
// execution of a task function.
//
// If it returns zero, the function does not get run at all.
//
// If it returns a duration greater than zero, the task function gets run once
// immediately and then again after the specified amount of time. At that point
// the Task re-invokes the schedule function and repeats the same logic.
//
// If ErrSkip is returned, the immediate execution of the task function gets
// skipped, and it will only be possibly executed after the returned interval.
//
// If any other error is returned, the task won't execute the function, however
// if the returned interval is greater than zero it will re-try to run the
// schedule function after that amount of time.
type Schedule func(err error) (time.Duration, error)

// ErrSkip is a special error that may be returned by a Schedule function to
// mean to skip a particular execution of the task function, and just wait the
// returned interval before re-evaluating.
var ErrSkip = errors.Errorf("skip execution of task function")

// Every returns a Schedule that always returns the given time interval.
func Every(interval time.Duration, options ...EveryOption) Schedule {
	opt := new(every)
	for _, option := range options {
		option(opt)
	}
	first := true
	return func(err error) (time.Duration, error) {
		// If we encounter a backoff error in an every request, then we can't
		// handle it and just treat it like a nil error.
		if err == ErrBackoff {
			err = nil
		}

		if first && opt.skipFirst {
			err = ErrSkip
		}
		first = false
		return interval, err
	}
}

// Daily is a convenience for creating a schedule that runs once a day.
func Daily(options ...EveryOption) Schedule {
	return Every(24*time.Hour, options...)
}

// EveryOptions represents a way to set optional values to a every option.
// The EveryOptions shows what options are available to change.
type EveryOptions interface {
	SetSkipFirst(bool)
}

// SkipFirst is an option for the Every schedule that will make the schedule
// skip the very first invocation of the task function.
var SkipFirst = func(every EveryOptions) { every.SetSkipFirst(true) }

// EveryOption captures a tweak that can be applied to the Every schedule.
type EveryOption func(EveryOptions)

// Captures options for the Every schedule.
type every struct {
	skipFirst bool // If true, return ErrSkip at the very first execution
}

func (e *every) SetSkipFirst(b bool) {
	e.skipFirst = b
}

// ErrBackoff is a special error that may be returned by a Schedule function to
// mean backoff the interval, before re-evaluating the interval of the next
// execution.
var ErrBackoff = errors.Errorf("skip execution of task function")

// Backoff returns a Schedule that will attempt to backoff if there is a error
// passed.
func Backoff(interval time.Duration, options ...BackoffOption) Schedule {
	opt := new(backoff)
	for _, option := range options {
		option(opt)
	}
	var (
		amount           = 0
		originalInterval = interval
	)
	return func(err error) (time.Duration, error) {
		// Attempt to handle the backing off
		if err == nil {
			amount = 0
			interval = originalInterval
		} else if err == ErrBackoff && opt.backoff != nil {
			amount++
			return opt.backoff(amount, originalInterval), nil
		}
		return interval, err
	}
}

// BackoffFunc is a type that represents a way to describe what the interval
// time should be for each backoff request. The amount is used to say how many
// iterations have occurred since the backoff was requested.
type BackoffFunc func(n int, t time.Duration) time.Duration

// BackoffOptions represents a way to set optional values to a backoff option.
// The BackoffOptions shows what options are available to change.
type BackoffOptions interface {
	SetBackoff(BackoffFunc)
}

// BackoffOption captures a tweak that can be applied to the Backoff schedule.
type BackoffOption func(BackoffOptions)

// Captures options for the Backoff schedule.
type backoff struct {
	backoff BackoffFunc
}

func (e *backoff) SetBackoff(f BackoffFunc) {
	e.backoff = f
}

// Linear describes a backoff function that grows linearly with time.
var Linear = func(backoff BackoffOptions) {
	backoff.SetBackoff(func(n int, t time.Duration) time.Duration {
		return t * time.Duration(n)
	})
}
