package guard

import (
	"errors"
	"testing"
	"time"
)

//go:generate mockgen -package guard -destination task_mock_test.go github.com/spoke-d/task/guard Task,Guest

func stopGuard(g *Guard) error {
	if err := g.Kill(); err != nil {
		return err
	}
	return g.Wait()
}

func assertGuardStopped(t *testing.T, g *Guard) {
	t.Helper()

	if err := stopGuard(g); err != nil {
		t.Fatal(err)
	}
}

func assertLocked(t *testing.T, guest Guest) {
	t.Helper()

	visited := make(chan error)
	abort := make(chan struct{})
	go func() {
		visited <- guest.Visit(func() error {
			return errors.New("bad")
		}, abort)
	}()

	delay := time.After(time.Millisecond * 50)
	for {
		select {
		case <-delay:
			delay = nil
			close(abort)
		case err := <-visited:
			if expected, actual := ErrAborted, err; expected != actual {
				t.Errorf("expected: %v, actual: %v", expected, actual)
			}
			return
		case <-time.After(time.Second * 10):
			t.Fatal("timed out")
		}
	}
}

func assertUnlocked(t *testing.T, guest Guest) {
	t.Helper()

	visited := make(chan error)
	go func() {
		visited <- guest.Visit(func() error {
			return errors.New("bad")
		}, nil)
	}()

	for {
		select {
		case err := <-visited:
			if expected, actual := "bad", err.Error(); expected != actual {
				t.Errorf("expected: %v, actual: %v", expected, actual)
			}
			return
		case <-time.After(time.Second * 10):
			t.Fatal("timed out")
		}
	}
}
