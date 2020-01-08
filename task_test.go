package task

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestTaskExecuteImmediately(t *testing.T) {
	f, wait := newFunc(t, 1)
	defer startTask(t, f, Every(time.Second))()
	wait(100 * time.Millisecond)
}

func TestTaskExecutePeriodically(t *testing.T) {
	f, wait := newFunc(t, 2)
	defer startTask(t, f, Every(250*time.Millisecond))()
	wait(100 * time.Millisecond)
	wait(400 * time.Millisecond)
}

func TestTaskExecuteTermination(t *testing.T) {
	f, wait := newFuncTerminates(t, 2)
	defer startTask(t, f, Every(250*time.Millisecond))()
	wait(100 * time.Millisecond)
	wait(400 * time.Millisecond)
	wait(800 * time.Millisecond)
}

func TestTaskReset(t *testing.T) {
	f, wait := newFunc(t, 3)
	stop, reset := Start(f, Every(250*time.Millisecond))
	defer stop(time.Second)

	wait(50 * time.Millisecond)
	reset()
	wait(50 * time.Millisecond)
	wait(400 * time.Millisecond)
}

func TestTaskZeroInterval(t *testing.T) {
	f, _ := newFunc(t, 0)
	defer startTask(t, f, Every(0*time.Millisecond))()

	time.Sleep(100 * time.Millisecond)
}

func TestTaskScheduleError(t *testing.T) {
	schedule := func(err error) (time.Duration, error) {
		return 0, errors.Errorf("bad")
	}
	f, _ := newFunc(t, 0)
	defer startTask(t, f, schedule)()

	time.Sleep(100 * time.Millisecond)
}

func TestTaskScheduleTemporaryError(t *testing.T) {
	errored := false
	schedule := func(err error) (time.Duration, error) {
		if !errored {
			errored = true
			return time.Millisecond, errors.Errorf("bad")
		}
		return time.Second, nil
	}
	f, wait := newFunc(t, 1)
	defer startTask(t, f, schedule)()

	wait(50 * time.Millisecond)
}

func TestTaskSkipFirst(t *testing.T) {
	i := 0
	f := func(context.Context) error {
		i++
		return nil
	}
	defer startTask(t, f, Every(30*time.Millisecond, SkipFirst))()
	time.Sleep(40 * time.Millisecond)

	if expected, actual := 1, i; expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}
}

// Create a new task function that sends a notification to a channel every time
// it's run.
//
// Return the task function, along with a "wait" function which will block
// until one notification is received through such channel, or fails the test
// if no notification is received within the given timeout.
//
// The n parameter can be used to limit the number of times the task function
// is allowed run: when that number is reached the task function will trigger a
// test failure (zero means that the task function will make the test fail as
// soon as it is invoked).
func newFunc(t *testing.T, n int) (Func, func(time.Duration)) {
	t.Helper()

	i := 0
	notifications := make(chan struct{})
	f := func(context.Context) error {
		if i == n {
			t.Fatalf("task was supposed to be called at most %d times", n)
		}
		notifications <- struct{}{}
		i++
		return nil
	}
	wait := func(timeout time.Duration) {
		select {
		case <-notifications:
		case <-time.After(timeout):
			t.Fatalf("no notification received in %s", timeout)
		}
	}
	return f, wait
}

// Create a new task function that sends a notification to a channel every time
// it's run.
//
// Return the task function, along with a "wait" function which will block
// until one notification is received through such channel, or fails the test
// if too many notifications are received within the given timeout.
//
// The n parameter can be used to limit the number of times the task function
// is allowed run: when that number is reached the task function will trigger
// terminate error. This error ensures that the task can self terminate and no
// other invocations are run.
func newFuncTerminates(t *testing.T, n int) (Func, func(time.Duration)) {
	t.Helper()

	i := 0
	notifications := make(chan struct{})
	f := func(context.Context) error {
		if i == n {
			return ErrTerminate
		}
		notifications <- struct{}{}
		i++
		return nil
	}
	wait := func(timeout time.Duration) {
		select {
		case <-notifications:
			if i > n {
				t.Fatalf("unexpected notification received: %d", i)
			}
		case <-time.After(timeout):
			if i != n {
				t.Fatalf("no notification received in %s for %d", timeout, i)
			}
		}
	}
	return f, wait
}

// Convenience around Start which also makes sure that the stop function
// of the task actually terminates.
func startTask(t *testing.T, f Func, schedule Schedule) func() {
	t.Helper()

	stop, _ := Start(f, schedule)
	return func() {
		if err := stop(time.Second); err != nil {
			t.Fatal(err)
		}
	}
}
