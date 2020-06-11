package group

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestZero(t *testing.T) {
	var g Group

	res := make(chan error)
	go func() { res <- g.Run() }()

	select {
	case err := <-res:
		if err != nil {
			t.Errorf("%v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestOne(t *testing.T) {
	var (
		g Group

		myError = errors.New("foobar")
		res     = make(chan error)
	)

	g.Add(func(context.Context) error { return myError }, func(error) {})
	go func() { res <- g.Run() }()

	select {
	case err := <-res:
		if expected, actual := myError, err; expected != actual {
			t.Errorf("expected: %v, actual: %v", expected, actual)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestMany(t *testing.T) {
	var (
		g Group

		interrupt = errors.New("interrupt")
		cancel    = make(chan struct{})
		res       = make(chan error)
	)

	g.Add(func(context.Context) error { return interrupt }, func(error) {})
	g.Add(func(context.Context) error { <-cancel; return nil }, func(error) { close(cancel) })
	go func() { res <- g.Run() }()

	select {
	case err := <-res:
		if expected, actual := interrupt, err; expected != actual {
			t.Errorf("expected: %v, actual: %v", expected, actual)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}
