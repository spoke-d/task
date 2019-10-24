package guard

import (
	"errors"
	"sync"

	"github.com/spoke-d/task/tomb"
)

// ErrAborted is used to confirm clean termination of a blocking operation.
var ErrAborted = errors.New("guard operation aborted")

// ErrShutdown is used to report that the guard worker is shutting down.
var ErrShutdown = errors.New("guard worker shutting down")

// Guard coordinates between clients that access it as a Guard and as a Guest.
type Guard struct {
	tomb   *tomb.Tomb
	guards chan guard
	guests chan guest
}

// New returns a new, locked, guard. The caller is responsible for
// ensuring it somehow gets Kill()ed, and for handling any error returned by
// Wait().
func New() *Guard {
	guard := &Guard{
		tomb:   tomb.New(),
		guards: make(chan guard),
		guests: make(chan guest),
	}
	guard.tomb.Go(guard.loop)
	return guard
}

// Kill asks the guard to stop and returns immediately.
func (g *Guard) Kill() error {
	return g.tomb.Kill(nil)
}

// Wait waits for the Guard to complete and returns any error encountered when
// it was running or stopping.
func (g *Guard) Wait() error {
	return g.tomb.Wait()
}

// Unlock unblocks all visit calls.
func (g *Guard) Unlock() error {
	return g.allowGuests(true, nil)
}

// Lock blocks new visit calls, and waits for existing calls to
// complete; it will return ErrAborted if the supplied abort is closed
// before lock is complete. In this situation, the guard will
// remain closed to new visits, but may still be executing pre-existing
// ones; you need to wait for a lock to complete successfully before
// you can infer exclusive access.
func (g *Guard) Lock(abort <-chan struct{}) error {
	return g.allowGuests(false, abort)
}

// Visit waits until the guard is unlocked, then runs the supplied func.
// It will return ErrAborted if the supplied Abort is closed before the func
// is started.
func (g *Guard) Visit(visit func() error, abort <-chan struct{}) error {
	result := make(chan error)
	select {
	case <-g.tomb.Dying():
		return ErrShutdown
	case <-abort:
		return ErrAborted
	case g.guests <- guest{
		fn:     visit,
		result: result,
	}:
		return <-result
	}
}

// allowGuests communicates Guard-interface requests to the main loop.
func (g *Guard) allowGuests(allowGuests bool, abort <-chan struct{}) error {
	result := make(chan error)
	select {
	case <-g.tomb.Dying():
		return ErrShutdown
	case g.guards <- guard{
		allowGuests: allowGuests,
		abort:       abort,
		result:      result,
	}:
		return <-result
	}
}

// loop waits for a Guard to unlock the Guard, and then runs visit funcs in
// parallel until a Guard locks it down again; at which point, it waits for all
// outstanding visits to complete, and reverts to its original state.
func (g *Guard) loop() error {
	var active sync.WaitGroup
	defer active.Wait()

	// guestTickets will be set on Unlock and cleared at the start of lock.
	var guests <-chan guest
	for {
		select {
		case <-g.tomb.Dying():
			return tomb.ErrDying
		case t := <-guests:
			active.Add(1)
			go t.complete(active.Done)
		case t := <-g.guards:
			// guard ticket requests are idempotent; it's not worth building
			// the extra mechanism needed to (1) complain about abuse but
			// (2) remain comprehensible and functional in the face of aborted
			// Lockdowns.
			if t.allowGuests {
				guests = g.guests
			} else {
				guests = nil
			}
			go t.complete(active.Wait)
		}
	}
}

// guard communicates between the guard and the main loop.
type guard struct {
	allowGuests bool
	abort       <-chan struct{}
	result      chan<- error
}

// complete unconditionally sends a single value on guard.result; either nil
// (when the desired state is reached) or ErrAborted (when the ticket's Abort
// is closed). It should be called on its own goroutine.
func (g guard) complete(wait func()) {
	var result error
	defer func() {
		g.result <- result
	}()

	done := make(chan struct{})
	go func() {
		// If we're locking down, we should wait for all Visits to complete.
		// If not, Visits are already being accepted and we're already done.
		if !g.allowGuests {
			wait()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-g.abort:
		result = ErrAborted
	}
}

// guest communicates between the guard and the main loop.
type guest struct {
	fn     func() error
	result chan<- error
}

// complete unconditionally sends any error returned from the func, then
// calls the finished func. It should be called on its own goroutine.
func (g guest) complete(done func()) {
	defer done()
	g.result <- g.fn()
}
