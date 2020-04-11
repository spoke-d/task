package guard

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spoke-d/task/tomb"
)

func TestAbort(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	run := func() (Task, error) {
		t.Fatal("should be called")
		return nil, nil
	}

	abort := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)

		task, err := Occupy(guard, run, abort)
		if task != nil {
			t.Errorf("expected task to be nil, actual: %v", task)
		}
		if expected, actual := ErrAborted, errors.Cause(err); expected != actual {
			t.Errorf("expected err to be %v, actual: %v", expected, actual)
		}
	}()

	select {
	case <-done:
		t.Fatal("started early")
	case <-time.After(time.Millisecond * 50):
	}

	close(abort)
	select {
	case <-done:
	case <-time.After(time.Second * 50):
		t.Fatal("never cancelled")
	}
}

func TestStartError(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}

	run := func() (Task, error) {
		return nil, errors.New("bad")
	}
	task, err := Occupy(guard, run, nil)
	if task != nil {
		t.Errorf("expected task to be nil, actual: %v", task)
	}
	if expected, actual := "bad", err.Error(); expected != actual {
		t.Errorf("expected err to be %v, actual: %v", expected, actual)
	}

	err = guard.Lock(nil)
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}
	assertLocked(t, guard)
}

func TestStartSuccess(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}

	stubTask := newTask()
	defer killTask(t, stubTask)

	task, err := Occupy(guard, func() (Task, error) {
		return stubTask, nil
	}, nil)
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}
	if expected, actual := stubTask, task; expected != actual {
		t.Errorf("expected %v, actual: %v", expected, actual)
	}

	locked := make(chan error, 1)
	go func() {
		locked <- guard.Lock(nil)
	}()
	select {
	case err := <-locked:
		t.Fatalf("unexpected Lockdown result: %v", err)
	case <-time.After(time.Millisecond * 50):
	}

	killTask(t, stubTask)
	select {
	case err := <-locked:
		if err != nil {
			t.Errorf("expected err to be nil, actual: %v", err)
		}
	case <-time.After(time.Second * 20):
		t.Fatal("visit never completed")
	}
}

func TestStartSkip(t *testing.T) {
	guard := New()
	defer assertGuardStopped(t, guard)

	err := guard.Unlock()
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}

	task, err := Occupy(guard, func() (Task, error) {
		return nil, ErrSkip
	}, nil)
	if err != nil {
		t.Errorf("expected err to be nil, actual: %v", err)
	}
	if expected, actual := true, task == nil; expected != actual {
		t.Errorf("expected %v, actual: %v", expected, actual)
	}
}

type task struct {
	tomb *tomb.Tomb
}

func newTask() *task {
	t := &task{
		tomb: tomb.New(false),
	}
	t.tomb.Go(func(context.Context) error {
		<-t.tomb.Dying()
		return nil
	})
	return t
}

func (t *task) Kill() {
	t.tomb.Kill(nil)
}

func (t *task) Wait() error {
	return t.tomb.Wait()
}

func killTask(t *testing.T, tsk *task) {
	tsk.Kill()

	wait := make(chan error, 1)
	go func() {
		wait <- tsk.Wait()
	}()
	select {
	case err := <-wait:
		if err != nil {
			t.Errorf("expected err to be nil, actual: %v", err)
		}
	case <-time.After(time.Second * 20):
		t.Fatal("timed out")
	}
}
