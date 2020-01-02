package task

import (
	"context"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	ok := make(chan struct{})
	f := func(context.Context) { close(ok) }

	stop, _ := Start(f, Every(time.Second))

	select {
	case <-ok:
	case <-time.After(time.Second):
		t.Fatal("test expired")
	}

	err := stop(time.Second)
	if err != nil {
		t.Errorf("expected err to be nil")
	}
}
