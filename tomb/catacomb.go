package tomb

import (
	"sync"

	"github.com/pkg/errors"
)

// Catacomb is a variant of Tomb with its own internal goroutine, designed
// for coordinating the lifetimes of tombs needed by a single parent.
type Catacomb struct {
	tomb  *Tomb
	wg    sync.WaitGroup
	tombs chan *Tomb
}

// NewCatacomb creates a new catacomb to process tombs.
func NewCatacomb() *Catacomb {
	return &Catacomb{
		tomb:  New(false),
		tombs: make(chan *Tomb),
	}
}

// Add causes the supplied tomb's lifetime to be bound to the catacomb's,
// relieving the client of responsibility for Kill()ing it and Wait()ing for an
// error, *whether or not this method succeeds*. If the method returns an error,
// it always indicates that the catacomb is shutting down; the value will either
// be the error from the (now-stopped) tomb, or catacomb.ErrDying().
//
// If the tomb completes without error, the catacomb will continue unaffected;
// otherwise the catacomb's tomb will be killed with the returned error. This
// allows clients to freely Kill() tombs that have been Add()ed; any errors
// encountered will still kill the catacomb, so the tombs stay under control
// until the last moment, and so can be managed pretty casually once they've
// been added.
//
// Don't try to add a tomb to its own catacomb; that'll deadlock the shutdown
// procedure. I don't think there's much we can do about that.
func (c *Catacomb) Add(t *Tomb) error {
	select {
	case <-c.tomb.Dying():
		if err := Stop(t); err != nil {
			return errors.WithStack(err)
		}
		return c.ErrDying()
	case c.tombs <- t:
		// Note that we don't need to wait for confirmation here. This depends
		// on the catacomb.wg.Add() for the listen loop, which ensures the wg
		// won't complete until no more adds can be received.
		return nil
	}
}

// Dying returns a channel that will be closed when Kill is called.
func (c *Catacomb) Dying() <-chan struct{} {
	return c.tomb.Dying()
}

// Dead returns a channel that will be closed when Invoke has completed (and
// thus when subsequent calls to Wait() are known not to block).
func (c *Catacomb) Dead() <-chan struct{} {
	return c.tomb.Dead()
}

// Wait blocks until Invoke completes, and returns the first non-nil and
// non-tomb.ErrDying error passed to Kill before Invoke finished.
func (c *Catacomb) Wait() error {
	return c.tomb.Wait()
}

// Err returns the reason for the tomb death provided via Kill
// or Killf, or ErrStillAlive when the tomb is still alive.
func (c *Catacomb) Err() error {
	return c.tomb.Err()
}

// ErrDying returns an error that can be used to Kill *this* catacomb without
// overwriting nil errors. It should only be used when the catacomb is already
// known to be dying; calling this method at any other time will return a
// different error, indicating client misuse.
func (c *Catacomb) ErrDying() error {
	select {
	case <-c.tomb.Dying():
		return ErrDying
	default:
		return errors.New("bad catacomb ErrDying: still alive")
	}
}

// Kill will kill the Catacomb's internal tomb with the supplied error, or one
// derived from it.
//
// It's always safe to call Kill, but errors passed to Kill after the catacomb
// is dead will be ignored.
func (c *Catacomb) Kill(err error) error {
	return c.tomb.Kill(err)
}

// add starts two goroutines that (1) kill the catacomb's tomb with any
// error encountered by the tomb; and (2) kill the tomb when the
// catacomb starts dying.
func (c *Catacomb) add(t *Tomb) {
	c.wg.Add(2)

	done := make(chan struct{})
	go func() {
		defer c.wg.Done()
		defer close(done)

		if err := t.Wait(); err != nil {
			c.Kill(err)
		}
	}()

	go func() {
		defer c.wg.Done()
		select {
		case <-c.tomb.Dying():
			Stop(t)
		case <-done:
		}
	}()
}
