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

type everyOption interface {
	setSkipFirst(bool)
}

// SkipFirst is an option for the Every schedule that will make the schedule
// skip the very first invocation of the task function.
var SkipFirst = func(every everyOption) { every.setSkipFirst(true) }

// EveryOption captures a tweak that can be applied to the Every schedule.
type EveryOption func(everyOption)

// Captures options for the Every schedule.
type every struct {
	skipFirst bool // If true, return ErrSkip at the very first execution
}

func (e *every) setSkipFirst(b bool) {
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
		amount           = 1
		originalInterval = interval
	)
	return func(err error) (time.Duration, error) {
		// Attempt to handle the backing off
		if err == nil {
			amount = 1
			interval = originalInterval
		} else if err == ErrBackoff && opt.backoff != nil {
			interval = opt.backoff(amount, originalInterval)
			amount++
		}
		return interval, err
	}
}

type fnBackoff func(n int, t time.Duration) time.Duration

type backoffOption interface {
	setBackoff(fnBackoff)
}

// BackoffOption captures a tweak that can be applied to the Backoff schedule.
type BackoffOption func(backoffOption)

// Captures options for the Backoff schedule.
type backoff struct {
	backoff fnBackoff
}

func (e *backoff) setBackoff(f fnBackoff) {
	e.backoff = f
}

// Linear describes a backoff function that grows linearly with time.
var Linear = func(backoff backoffOption) {
	backoff.setBackoff(func(n int, t time.Duration) time.Duration {
		return t * time.Duration(n)
	})
}
