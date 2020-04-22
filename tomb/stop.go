package tomb

import "github.com/pkg/errors"

// CancellableTask defines a task that can be waited on and also cancelled via
// a kill method.
type CancellableTask interface {
	// Kill asks the task to stop and returns immediately.
	Kill(error) error

	// Wait waits for the task to complete and returns any error encountered
	// when it was running or stopping.
	Wait() error
}

// Stop kills the given tomb and waits for it to complete.
func Stop(t CancellableTask) error {
	if err := t.Kill(nil); err != nil {
		return errors.WithStack(err)
	}
	return t.Wait()
}
