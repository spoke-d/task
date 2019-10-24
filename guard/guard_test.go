package guard

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestStoppedUnlock(t *testing.T) {
	guard := New()
	assertGuardStopped(t, guard)

	err := guard.Unlock()
	if expected, actual := ErrShutdown, err; expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func TestStoppedLock(t *testing.T) {
	guard := New()
	assertGuardStopped(t, guard)

	err := guard.Lock(nil)
	if expected, actual := ErrShutdown, err; expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func TestStoppedVisit(t *testing.T) {
	guard := New()
	assertGuardStopped(t, guard)

	err := guard.Visit(func() error {
		t.Fatalf("error if called")
		return nil
	}, nil)
	if expected, actual := ErrShutdown, err; expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func TestStartsLocked(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	assertLocked(t, guard)
}

func TestLock(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Lock(nil)
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}
	assertLocked(t, guard)
}

func TestUnlock(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}
	assertUnlocked(t, guard)
}

func TestMultipleUnlock(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	for i := 0; i < 3; i++ {
		err := guard.Unlock()
		if err != nil {
			t.Errorf("expected err to be nil, actual: %v", err)
		}
	}
	assertUnlocked(t, guard)
}

func TestMultipleLock(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}

	for i := 0; i < 3; i++ {
		err := guard.Lock(nil)
		if err != nil {
			t.Errorf("expected err to be nil, actual: %v", err)
		}
	}
	assertLocked(t, guard)
}

func TestVisitError(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}

	err = guard.Visit(func() error {
		return errors.New("bad")
	}, nil)
	if expected, actual := "bad", err.Error(); expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func TestVisitSuccess(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}

	err = guard.Visit(func() error {
		return nil
	}, nil)
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}
}

func TestConcurrentVisit(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}

	const count = 10
	var started sync.WaitGroup

	finished := make(chan int, count)
	unblocked := make(chan struct{})

	for i := 0; i < count; i++ {
		started.Add(1)
		go func(i int) {
			err := guard.Visit(func() error {
				started.Done()
				<-unblocked
				return nil
			}, nil)
			if err != nil {
				t.Errorf("expected err to be nil, actual: %v", err)
			}
			finished <- i
		}(i)
	}
	started.Wait()

	assertUnlocked(t, guard)

	close(unblocked)

	timeout := time.After(time.Second * 10)
	seen := make(map[int]struct{})
	for i := 0; i < count; i++ {
		select {
		case f := <-finished:
			seen[f] = struct{}{}
		case <-timeout:
			t.Errorf("timed out waiting for %dth result", i)
		}
	}
	if expected, actual := count, len(seen); expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}
