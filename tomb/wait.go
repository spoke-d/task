package tomb

import (
	"time"

	"github.com/pkg/errors"
)

// Waitable defines a type that states if it's dead or not.
type Waitable interface {
	// Dying returns the channel that can be used to wait until
	// t.Kill is called.
	Dying() <-chan struct{}
}

// Wait for a Waitable to die.
func Wait(waitable Waitable, duration time.Duration) error {
	select {
	case <-waitable.Dying():
	case <-time.After(duration):
		return errors.Errorf("timed out waiting for death")
	}
	return nil
}
