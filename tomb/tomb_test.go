package tomb

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func nothing(context.Context) error { return nil }

func TestNewTomb(t *testing.T) {
	tb := New(false)
	checkState(t, tb, false, false, ErrStillAlive)
}

func TestGo(t *testing.T) {
	tb := New(false)

	alive := make(chan bool)
	tb.Go(func(context.Context) error {
		alive <- true
		tb.Go(func(context.Context) error {
			alive <- true
			<-tb.Dying()
			return nil
		})
		<-tb.Dying()
		return nil
	})
	<-alive
	<-alive

	checkState(t, tb, false, false, ErrStillAlive)

	tb.Kill(nil)
	tb.Wait()

	checkState(t, tb, true, true, nil)
}

func TestGoKeepAlive(t *testing.T) {
	tb := New(true)

	alive := make(chan bool)
	tb.Go(func(context.Context) error {
		alive <- true
		tb.Go(func(context.Context) error {
			alive <- true
			return nil
		})
		return nil
	})
	<-alive
	<-alive

	select {
	case <-tb.Dying():
		t.Fatal("should never go in to dying state")
	case <-time.After(time.Millisecond * 250):
	}

	checkState(t, tb, false, false, ErrStillAlive)

	tb.Kill(nil)
	tb.Wait()

	checkState(t, tb, true, true, nil)
}

func TestGoErr(t *testing.T) {
	first := errors.New("first error")
	second := errors.New("first error")

	tb := New(false)

	alive := make(chan bool)
	tb.Go(func(context.Context) error {
		alive <- true
		tb.Go(func(context.Context) error {
			alive <- true
			return first
		})
		<-tb.Dying()
		return second
	})

	<-alive
	<-alive

	tb.Wait()
	checkState(t, tb, true, true, first)
}

func TestGoPanic(t *testing.T) {
	// ErrDying being used properly, after a clean death.
	tb := New(false)
	tb.Go(nothing)
	tb.Wait()

	err := tb.Go(nothing)
	if expected, actual := "called after all goroutines terminated", errors.Cause(err).Error(); expected != actual {
		t.Fatal(err)
	}
	checkState(t, tb, true, true, nil)
}

func TestKill(t *testing.T) {
	// a nil reason flags the goroutine as dying
	tb := New(false)
	tb.Kill(nil)
	checkState(t, tb, true, false, nil)

	// a non-nil reason now will override Kill
	err := errors.New("some error")
	tb.Kill(err)
	checkState(t, tb, true, false, err)

	// another non-nil reason won't replace the first one
	tb.Kill(errors.New("ignore me"))
	checkState(t, tb, true, false, err)

	tb.Go(nothing)
	tb.Wait()
	checkState(t, tb, true, true, err)
}

func TestKillf(t *testing.T) {
	tb := New(false)

	err := tb.Killf("BO%s", "OM")
	if s := err.Error(); s != "BOOM" {
		t.Fatalf(`Killf("BO%s", "OM"): want "BOOM", got %q`, s, s)
	}
	checkState(t, tb, true, false, err)

	// another non-nil reason won't replace the first one
	tb.Killf("ignore me")
	checkState(t, tb, true, false, err)

	tb.Go(nothing)
	tb.Wait()
	checkState(t, tb, true, true, err)
}

func TestErrDying(t *testing.T) {
	// ErrDying being used properly, after a clean death.
	tb := New(false)
	tb.Kill(nil)
	tb.Kill(ErrDying)
	checkState(t, tb, true, false, nil)

	// ErrDying being used properly, after an errorful death.
	err := errors.New("some error")
	tb.Kill(err)
	tb.Kill(ErrDying)
	checkState(t, tb, true, false, err)

	// ErrDying being used badly, with an alive tomb.
	tb = New(false)
	err = tb.Kill(ErrDying)
	if expected, actual := "kill with dying while still alive", errors.Cause(err).Error(); expected != actual {
		t.Fatal(err)
	}

	checkState(t, tb, false, false, ErrStillAlive)
}

func TestKillErrStillAlivePanic(t *testing.T) {
	tb := New(false)
	err := tb.Kill(ErrStillAlive)
	if expected, actual := "kill with still alive", errors.Cause(err).Error(); expected != actual {
		t.Fatal(err)
	}

	checkState(t, tb, false, false, ErrStillAlive)
}

func checkState(t *testing.T, tb *Tomb, wantDying, wantDead bool, wantErr error) {
	select {
	case <-tb.Dying():
		if !wantDying {
			t.Error("<-Dying: should block")
		}
	default:
		if wantDying {
			t.Error("<-Dying: should not block")
		}
	}
	seemsDead := false
	select {
	case <-tb.Dead():
		if !wantDead {
			t.Error("<-Dead: should block")
		}
		seemsDead = true
	default:
		if wantDead {
			t.Error("<-Dead: should not block")
		}
	}
	if err := tb.Err(); err != wantErr {
		t.Errorf("Err: want %#v, got %#v", wantErr, err)
	}
	if wantDead && seemsDead {
		waitErr := tb.Wait()
		switch {
		case waitErr == ErrStillAlive:
			t.Errorf("Wait should not return ErrStillAlive")
		case !reflect.DeepEqual(waitErr, wantErr):
			t.Errorf("Wait: want %#v, got %#v", wantErr, waitErr)
		}
	}
}
