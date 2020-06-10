package guard

import (
	"context"
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
		visited <- guest.Visit(func(context.Context) error {
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
		visited <- guest.Visit(func(context.Context) error {
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

func startBlockingVisit(t *testing.T, guard *Guard) chan<- struct{} {
	t.Helper()

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}

	visitStarted := make(chan struct{}, 1)
	defer close(visitStarted)

	unblockVisit := make(chan struct{}, 1)
	go func() {
		err := guard.Visit(func(context.Context) error {
			visitStarted <- struct{}{}
			<-unblockVisit
			return nil
		}, nil)
		if err != nil {
			t.Errorf("expected err to be nil, actual: %v", err)
		}
	}()
	select {
	case <-visitStarted:
	case <-time.After(time.Second * 10):
		t.Fatal("visit never started")
	}
	return unblockVisit
}
