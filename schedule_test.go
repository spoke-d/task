package task

import (
	"testing"
	"time"
)

func TestEvery(t *testing.T) {
	fn := Every(time.Second)
	interval, err := fn(nil)
	if err != nil {
		t.Errorf("expected err to be nil")
	}
	if expected, actual := time.Second, interval; expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}
}

func TestEveryWithOption(t *testing.T) {
	fn := Every(time.Second, SkipFirst)
	interval, err := fn(nil)

	if expected, actual := ErrSkip, err; expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
	if expected, actual := time.Second, interval; expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}
}

func TestBackoff(t *testing.T) {
	fn := Backoff(time.Second)
	interval, err := fn(nil)
	if err != nil {
		t.Errorf("expected err to be nil")
	}
	if expected, actual := time.Second, interval; expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}
}

func TestBackoffWithOption(t *testing.T) {
	fn := Backoff(time.Second, Linear)

	interval, err := fn(nil)
	if expected, actual := true, err == nil; expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
	if expected, actual := time.Second, interval; expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}

	for i := 1; i < 5; i++ {
		interval, err = fn(ErrBackoff)
		if expected, actual := true, err == nil; expected != actual {
			t.Errorf("expected: %v, actual: %v", expected, actual)
		}
		if expected, actual := time.Second*time.Duration(i), interval; expected != actual {
			t.Errorf("expected: %d, actual: %d", expected, actual)
		}
	}

	interval, err = fn(nil)
	if expected, actual := true, err == nil; expected != actual {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
	if expected, actual := time.Second, interval; expected != actual {
		t.Errorf("expected: %d, actual: %d", expected, actual)
	}
}
