package tomb

import (
	"context"

	"github.com/pkg/errors"
)

// Invoke a set of Tombs to be overseen by a Catacomb.
func Invoke(tombs []*Tomb, fn Func) (*Catacomb, error) {
	catacomb := NewCatacomb()

	for _, tomb := range tombs {
		catacomb.add(tomb)
	}

	// This goroutine listens for added tombs until the catacomb is Killed.
	// We ensure the wg can't complete until we know no new tombs will be
	// added.
	catacomb.wg.Add(1)
	go func() {
		defer catacomb.wg.Done()
		for {
			select {
			case <-catacomb.tomb.Dying():
				return
			case w := <-catacomb.tombs:
				catacomb.add(w)
			}
		}
	}()

	if err := catacomb.tomb.Go(func(ctx context.Context) error {
		defer catacomb.wg.Wait()
		return fn(ctx)
	}); err != nil {
		for _, tomb := range tombs {
			err = errors.WithMessage(Stop(tomb), err.Error())
		}
		return nil, errors.WithStack(err)
	}

	return catacomb, nil
}
