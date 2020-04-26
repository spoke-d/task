package tomb

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestStartsAlive(t *testing.T) {
	t.Parallel()

	err := run(t, func(ctx context.Context) error {
		c := ctxCatacomb(t, ctx)
		assertNotDying(t, c)
		assertNotDead(t, c)
		return nil
	})
	if err != nil {
		t.Fatalf("expected err to be nil, %v", err)
	}
}

func TestKillClosesDying(t *testing.T) {
	t.Parallel()

	err := run(t, func(ctx context.Context) error {
		c := ctxCatacomb(t, ctx)
		c.Kill(nil)
		assertDying(t, c)
		return nil
	})
	if err != nil {
		t.Fatalf("expected err to be nil, %v", err)
	}
}

func TestKillDoesNotCloseDead(t *testing.T) {
	t.Parallel()

	err := run(t, func(ctx context.Context) error {
		c := ctxCatacomb(t, ctx)
		c.Kill(nil)
		assertNotDead(t, c)
		return nil
	})
	if err != nil {
		t.Fatalf("expected err to be nil, %v", err)
	}
}

func TestKillNonNilOverwritesNil(t *testing.T) {
	t.Parallel()

	second := errors.Errorf("boom")
	err := run(t, func(ctx context.Context) error {
		c := ctxCatacomb(t, ctx)
		c.Kill(nil)
		c.Kill(second)
		return nil
	})
	if expected, actual := "boom", err.Error(); expected != actual {
		t.Fatalf("expected: %v, actual: %v", expected, actual)
	}
}

func TestKillNonNilOverwritesNonNil(t *testing.T) {
	t.Parallel()

	first := errors.Errorf("boom")
	err := run(t, func(ctx context.Context) error {
		c := ctxCatacomb(t, ctx)
		c.Kill(first)
		c.Kill(nil)
		return nil
	})
	if expected, actual := "boom", err.Error(); expected != actual {
		t.Fatalf("expected: %v, actual: %v", expected, actual)
	}
}

func TestKillAliveErrDying(t *testing.T) {
	t.Parallel()

	var notDying error
	err := run(t, func(ctx context.Context) error {
		c := ctxCatacomb(t, ctx)
		notDying = c.ErrDying()
		c.Kill(notDying)
		return nil
	})
	if expected, actual := notDying, err; expected != actual {
		t.Fatalf("expected: %v, actual: %v", expected, actual)
	}
}

func run(t *testing.T, task func(ctx context.Context) error) error {
	t.Helper()

	c, err := Invoke(task)
	if err != nil {
		t.Fatalf("expected err to be nil, %v", err)
	}

	select {
	case <-c.Dead():
	case <-time.After(time.Millisecond * 250):
		t.Fatalf("timed out waiting for death")
	}
	return c.Wait()
}

func waitDying(t *testing.T, c *Catacomb) {
	t.Helper()

	if err := Wait(c, time.Millisecond*250); err != nil {
		t.Errorf("expected err to be nil, %v", err)
	}
}

func assertDying(t *testing.T, c *Catacomb) {
	t.Helper()

	select {
	case <-c.Dying():
	default:
		t.Fatalf("still alive")
	}
}

func assertNotDying(t *testing.T, c *Catacomb) {
	t.Helper()

	select {
	case <-c.Dying():
		t.Fatalf("already dying")
	default:
	}
}

func assertDead(t *testing.T, c *Catacomb) {
	t.Helper()

	select {
	case <-c.Dead():
	default:
		t.Fatalf("not dead")
	}
}

func assertNotDead(t *testing.T, c *Catacomb) {
	t.Helper()

	select {
	case <-c.Dead():
		t.Fatalf("already dead")
	default:
	}
}

func ctxCatacomb(t *testing.T, ctx context.Context) *Catacomb {
	t.Helper()

	src := ctx.Value(TombSource)
	if src == nil {
		t.Fatalf("expected valid tomb source")
	}
	if c, ok := src.(*Catacomb); ok {
		return c
	}
	t.Fatalf("unexpected tomb source, %T", src)
	return nil
}
