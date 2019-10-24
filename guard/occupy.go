package guard

import "github.com/juju/errors"

// Task describes any type whose validity and/or activity is bounded
// in time. Most frequently, they will represent the duration of some
// task or tasks running on internal goroutines, but it's possible and
// rational to use them to represent any resource that might become
// invalid.
//
// Task implementations must be goroutine-safe.
type Task interface {

	// Wait waits for the Task to complete and returns any
	// error encountered when it was running or stopping.
	Wait() error
}

// StartFunc starts a Task.
type StartFunc func() (Task, error)

// Guest allows clients to Visit a guard when it's unlocked; that is, to
// get non-exclusive access to whatever resource is being protected for the
// duration of the supplied Visit func.
type Guest interface {

	// Visit waits until the guard is unlocked, then runs the supplied
	// Visit func. It will return ErrAborted if the supplied Abort is closed
	// before the Visit is started.
	Visit(fn func() error, abort <-chan struct{}) error
}

// Occupy launches a Visit to guard that creates a task and holds the
// visit open until the task completes. Like most funcs that return any
// task, the caller takes responsibility for its lifetime; be aware that
// the responsibility is especially heavy here, because failure to clean up
// the task will block cleanup of the guard.
//
// This may sound scary, but the alternative is to have multiple components
// "responsible for" a single task's lifetime -- and guard itself would
// have to grow new concerns, of understanding and managing Tasks --
// and that scenario ends up much worse.
func Occupy(guard Guest, start StartFunc, abort <-chan struct{}) (Task, error) {
	// Create two channels to communicate success and failure of task
	// creation; and a task-running func that sends on exactly one
	// of them, and returns only when (1) a value has been sent and (2)
	// no task is running. Note especially that it always returns nil.
	started := make(chan Task, 1)
	failed := make(chan error, 1)

	task := func() error {
		tsk, err := start()
		if err != nil {
			failed <- err
			return nil
		}
		started <- tsk
		return tsk.Wait()
	}

	// Start a goroutine to run the task func inside the guard. If
	// this operation succeeds, we must inspect started and failed to
	// determine what actually happened; but if it fails, we can be
	// confident that the task (which never fails) did not run, and can
	// therefore return the failure without waiting further.
	finished := make(chan error, 1)
	go func() {
		finished <- guard.Visit(task, abort)
	}()

	// Watch all these channels to figure out what happened and inform
	// the client. A nil error from finished indicates that there will
	// be some value waiting on one of the other channels.
	for {
		select {
		case err := <-finished:
			if err != nil {
				return nil, errors.Trace(err)
			}
		case err := <-failed:
			return nil, errors.Trace(err)
		case tsk := <-started:
			return tsk, nil
		}
	}
}
