package task

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestGroupLen(t *testing.T) {
	group := NewGroup()
	defer group.Kill()

	ok := make(chan struct{})
	f := func(context.Context) { close(ok) }

	group.Add(f, Every(time.Second))
	group.Add(f, Every(time.Minute))

	if expected, actual := 2, group.Len(); expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}
}

func TestGroupStart(t *testing.T) {
	group := NewGroup()
	defer group.Kill()

	ok := make(chan struct{})
	f := func(context.Context) { close(ok) }

	group.Add(f, Every(time.Second*10))
	group.Start()

	select {
	case <-ok:
	case <-time.After(time.Second):
		t.Fatal("test expired")
	}

	err := group.Stop(time.Second)
	if err != nil {
		t.Errorf("expected err to be nil")
	}
}

func TestGroupStop(t *testing.T) {
	group := NewGroup()
	defer group.Kill()

	ok := make(chan struct{})
	f := func(context.Context) {
		ok <- struct{}{}
		<-ok
	}

	group.Add(f, Every(time.Second))
	err := group.Start()
	if err != nil {
		t.Errorf("expected err not to be nil")
	}

	select {
	case <-ok:
	case <-time.After(time.Second):
		t.Fatal("test expired")
	}

	time.AfterFunc(time.Millisecond, func() {
		close(ok)
	})

	err = group.Stop(time.Second)
	if err != nil {
		t.Errorf("expected err not to be nil")
	}
}

func TestGroupStopWithVisitBlocking(t *testing.T) {
	group := NewGroup()
	defer group.Kill()

	ok := make(chan struct{})
	defer close(ok)

	f := func(context.Context) {
		ok <- struct{}{}
		<-ok
	}

	group.Add(f, Every(time.Second))
	err := group.Start()
	if err != nil {
		t.Errorf("expected err not to be nil")
	}

	select {
	case <-ok:
	case <-time.After(time.Second):
		t.Fatal("test expired")
	}

	err = group.Stop(time.Millisecond)
	if err == nil {
		t.Errorf("expected err not to be nil")
	}
	if expected, actual := "tasks 1 are still running", errors.Cause(err).Error(); expected != actual {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}
}

func TestGroupStopStart(t *testing.T) {
	group := NewGroup()
	defer group.Kill()

	ok := make(chan struct{})
	defer close(ok)

	f := func(context.Context) {
		ok <- struct{}{}

	}

	group.Add(f, Every(time.Second))

	for i := 0; i < 5; i++ {
		err := group.Start()
		if err != nil {
			t.Errorf("expected err not to be nil")
		}

		select {
		case <-ok:
		case <-time.After(time.Second):
			t.Fatal("test expired")
		}

		err = group.Stop(time.Millisecond)
		if err != nil {
			t.Errorf("expected err not to be nil")
		}
	}
}

func TestGroupStopWithNoStart(t *testing.T) {
	group := NewGroup()
	defer group.Kill()

	err := group.Stop(time.Second)
	if err != nil {
		t.Errorf("expected err to be nil")
	}
}
